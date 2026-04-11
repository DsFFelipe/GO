package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

// Dados representa a estrutura JSON enviada pelos sensores via UDP.
// A serialização utiliza tags json para mapeamento direto entre os campos
// e a representação textual transmitida na rede.
type Dados struct {
	ID         string `json:"id"`
	Tipo       string `json:"tipo"`
	Valor      int    `json:"valor"`
	Localidade string `json:"localidade"`
}

// ComandoAtuador é a estrutura de controle interno utilizada para
// roteamento de ordens dentro do pipeline de concorrência do servidor.
// Separa a lógica de negócio (qual atuador e qual ação) do transporte de rede.
type ComandoAtuador struct {
	Alvo string // Identificador lógico do dispositivo: "BARREIRA" ou "ALARME"
	Acao string // Mensagem textual a ser transmitida ao atuador: "ABRIR", "LIGAR_ALARME", etc.
}

// Variáveis de estado global protegidas por Mutex.
// O estado é mantido em memória volátil; não há persistência em disco.
var (
	barreiraAberta bool       = true  // Estado inicial assumido: barreira aberta (seguro por padrão)
	alarmeLigado   bool       = false // Estado inicial assumido: alarme silencioso
	mu             sync.Mutex         // Protege o acesso concorrente a barreiraAberta e alarmeLigado

	// Tabela de roteamento dinâmica: mapeia a conexão TCP estabelecida com o atuador
	// ao seu tipo lógico. Utilizada pelo broker para saber para qual socket enviar cada comando.
	// Acesso concorrente: goroutine de registro (recebeAtuadores) e goroutine de envio (enviaComandosParaAtuadores).
	atuadoresAtivos = make(map[net.Conn]string)
	muAtuadores     sync.Mutex // Mutex específico para proteger o mapa atuadoresAtivos
)

// main configura os canais de comunicação e dispara as goroutines que compõem
// a arquitetura de pipeline do servidor. O uso de select {} vazio bloqueia
// a thread principal permanentemente, mantendo o processo em execução.
func main() {
	// Canal para propagar dados dos sensores (UDP) para o cliente monitor (UDP).
	// Capacidade 0 força sincronização estrita entre produtor e consumidor.
	sensorParaClienteChan := make(chan []byte)

	// Canal com buffer de 100 elementos para o pipeline de decisão.
	// O buffer evita bloqueios momentâneos na receção UDP caso as goroutines
	// de processamento estejam ocupadas com operações de I/O.
	DadosSensor := make(chan []byte, 100)

	// Canal para comandos direcionados aos atuadores.
	// Sem buffer; o broker consumirá assim que disponível.
	chAtuadores := make(chan ComandoAtuador)

	// Canal para transporte de mensagens de estado (JSON) vindas dos atuadores
	// que serão reencaminhadas via TCP para o cliente monitor.
	estadoAtuadorChan := make(chan []byte)

	// Dispara goroutines independentes para cada responsabilidade do servidor.
	go recebecliente(chAtuadores)                       // Escuta comandos manuais (TCP) do cliente terminal.
	go recebesensor(sensorParaClienteChan, DadosSensor) // Escuta telemetria UDP dos sensores.
	go enviacliente(sensorParaClienteChan)              // Reencaminha telemetria UDP para o monitor.

	// Pool de threads para processamento de decisão baseado em regras.
	// A quantidade de workers (3) foi escolhida para balancear concorrência
	// sem sobrecarregar o scheduler com goroutines excessivas para a carga esperada.
	for i := 0; i < 3; i++ {
		go processarDecisao(DadosSensor, chAtuadores)
	}

	go recebeAtuadores(estadoAtuadorChan)      // Gerencia conexões TCP dos atuadores e extrai handshake/estados.
	go enviaComandosParaAtuadores(chAtuadores) // Broker: roteia comandos para os atuadores corretos.
	go enviaEstadoTCP(estadoAtuadorChan)       // Encaminha estados dos atuadores para o monitor (TCP).

	// Bloqueia a goroutine principal indefinidamente.
	// Sem esta instrução, o programa terminaria imediatamente após iniciar as goroutines.
	select {}
}

// recebeAtuadores implementa o servidor TCP que aguarda conexões dos dispositivos atuadores.
// Cada atuador envia inicialmente uma linha de registo (ex: "REGISTRO:BARREIRA\n")
// e, em seguida, permanece conectado enviando atualizações de estado periodicamente.
// O delimitador '\n' é usado como framing de mensagens sobre o stream TCP.
func recebeAtuadores(chEstado chan<- []byte) {
	port := os.Getenv("ATUADOR_PORT")
	if port == "" {
		port = "8082" // Porta padrão para comunicação com atuadores
	}

	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Printf("Erro no Listen TCP Atuadores: %v\n", err)
		return
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			// Erros temporários são ignorados; o loop continua aceitando novas conexões.
			continue
		}

		// Cada conexão é tratada em uma goroutine dedicada para evitar bloqueio do Accept.
		go func(c net.Conn) {
			// bufio.Reader permite leitura eficiente linha a linha com buffer interno.
			reader := bufio.NewReader(c)

			// Protocolo de Handshake: a primeira linha enviada identifica o tipo do atuador.
			linha, err := reader.ReadString('\n')
			if err != nil {
				c.Close()
				return
			}

			identificacao := strings.TrimSpace(linha)
			tipo := "DESCONHECIDO"
			if identificacao == "REGISTRO:BARREIRA" {
				tipo = "BARREIRA"
			} else if identificacao == "REGISTRO:ALARME" {
				tipo = "ALARME"
			}

			// Regista a conexão na tabela de roteamento para que comandos possam ser encaminhados.
			muAtuadores.Lock()
			atuadoresAtivos[c] = tipo
			muAtuadores.Unlock()

			// Loop de escuta contínua: aguarda atualizações de estado enviadas pelo atuador.
			// Cada mensagem é delimitada por '\n' (conforme enviado pelo atuador).
			for {
				pacoteEstado, err := reader.ReadBytes('\n')
				if err != nil {
					// Em caso de erro (desconexão ou timeout), remove a entrada do mapa.
					c.Close()
					muAtuadores.Lock()
					delete(atuadoresAtivos, c)
					muAtuadores.Unlock()
					break
				}
				// Encaminha o estado recebido para o pipeline de notificação ao monitor.
				chEstado <- pacoteEstado
			}
		}(conn)
	}
}

// enviaEstadoTCP atua como cliente TCP para enviar atualizações de estado dos atuadores
// ao processo monitor (que roda na interface gráfica/terminal).
// Utiliza conexões efêmeras (abre, escreve, fecha) para cada mensagem.
// Este padrão push-based garante que o monitor tenha a visão mais recente do hardware.
func enviaEstadoTCP(ch <-chan []byte) {
	clienteTCP := os.Getenv("CLIENTE_TCP_ADDR")
	if clienteTCP == "" {
		clienteTCP = "192.168.0.118:8084" // Endereço do monitor em modo escuta TCP
	}

	for msg := range ch {
		// DialTimeout evita que a goroutine fique bloqueada indefinidamente
		// caso o monitor esteja offline ou inacessível.
		conn, err := net.DialTimeout("tcp", clienteTCP, 2*time.Second)
		if err == nil {
			// SetWriteDeadline protege contra buffers de rede congestionados.
			conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
			conn.Write(msg)
			conn.Close() // Fecha imediatamente após envio; não mantém conexão persistente.
		}
		// Em caso de falha, a mensagem é silenciosamente descartada.
		// A arquitetura tolera perda eventual de atualizações de estado.
	}
}

// ==========================================
// BROKER DE ATUADORES COM ROTEAMENTO (TCP: 8082)
// ==========================================

// enviaComandosParaAtuadores é o broker de roteamento de comandos.
// Consome mensagens do canal ch e as encaminha para todas as conexões
// cujo tipo corresponda ao Alvo especificado no comando.
// Esta lógica suporta múltiplos atuadores do mesmo tipo (ex: várias barreiras),
// embora o mapeamento atual suponha apenas um por tipo.
func enviaComandosParaAtuadores(ch <-chan ComandoAtuador) {
	for comando := range ch {
		muAtuadores.Lock()
		// Itera sobre o snapshot atual das conexões ativas.
		for conn, tipo := range atuadoresAtivos {
			// Filtro de roteamento baseado no tipo registado durante o handshake.
			if tipo == comando.Alvo {
				// Deadline de escrita para evitar bloqueio em sockets problemáticos.
				conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
				_, err := conn.Write([]byte(comando.Acao))
				if err != nil {
					// Se a escrita falhar, assume-se que o atuador está morto/desconectado.
					// Remove a conexão do mapa e fecha o socket.
					fmt.Printf("Atuador desconectado (falha de escrita): %s\n", conn.RemoteAddr().String())
					conn.Close()
					delete(atuadoresAtivos, conn)
				}
			}
		}
		muAtuadores.Unlock()
	}
}

// ==========================================
// LÓGICA DE SENSORES
// ==========================================
// processarDecisao implementa a lógica de negócio reativa baseada nos dados dos sensores.
// É executada por múltiplas goroutines (pool de threads) que consomem do mesmo canal.
// O uso de Mutex protege a leitura/escrita das variáveis de estado globais.
// As regras definidas:
// - Sensor Geiger > 95 e barreira aberta -> fecha barreira.
// - Sensor Geiger < 10 e barreira fechada -> abre barreira.
// - Temperatura reator > 600 e alarme desligado -> liga alarme.
// - Temperatura reator <= 500 e alarme ligado -> desliga alarme.
func processarDecisao(chSensor <-chan []byte, chAtuador chan<- ComandoAtuador) {
	for {
		rawBytes := <-chSensor

		var d Dados
		err := json.Unmarshal(rawBytes, &d)
		if err != nil {
			// Ignora pacotes malformados (ruído de rede ou erro de serialização).
			continue
		}

		// Seção crítica: acesso às variáveis barreiraAberta e alarmeLigado.
		mu.Lock()

		if d.Tipo == "Geiger" {
			// Lógica com histerese simples para evitar oscilações.
			if d.Valor > 95 && barreiraAberta {
				barreiraAberta = false
				fmt.Println("ALERTA: Nível crítico de radiação! Fechando barreira.")
				chAtuador <- ComandoAtuador{Alvo: "BARREIRA", Acao: "FECHAR"}
			} else if d.Valor < 10 && !barreiraAberta {
				barreiraAberta = true
				fmt.Println("STATUS: Nível seguro. Abrindo barreira.")
				chAtuador <- ComandoAtuador{Alvo: "BARREIRA", Acao: "ABRIR"}
			}

		} else if d.Tipo == "temperatura_reator" {
			if d.Valor > 600 && !alarmeLigado {
				alarmeLigado = true
				fmt.Println("ALERTA CRÍTICO: Temperatura do reator excedeu o limite! Acionando alarme.")
				chAtuador <- ComandoAtuador{Alvo: "ALARME", Acao: "LIGAR_ALARME"}
			} else if d.Valor <= 500 && alarmeLigado {
				alarmeLigado = false
				fmt.Println("STATUS: Temperatura do reator estabilizada. Desligando alarme.")
				chAtuador <- ComandoAtuador{Alvo: "ALARME", Acao: "DESLIGAR_ALARME"}
			}
		}
		mu.Unlock()
	}
}

// recebesensor abre um socket UDP para escuta de pacotes enviados pelos sensores.
// O UDP é utilizado devido à natureza de transmissão contínua e tolerância a perdas
// em dados de telemetria (melhor esforço).
// Cada pacote recebido é replicado para dois canais:
// - DadosSensor: para processamento interno (decisão).
// - sensorParaClienteChan: para retransmissão ao monitor (interface visual).
func recebesensor(chCliente chan<- []byte, DadosSensor chan<- []byte) {
	port := os.Getenv("SENSOR_PORT")
	if port == "" {
		port = "8080" // Porta padrão para receção de dados de sensores
	}

	endr, _ := net.ResolveUDPAddr("udp", "0.0.0.0:"+port)
	conn, _ := net.ListenUDP("udp", endr)
	defer conn.Close()
	fmt.Println("Servidor aguardando dados do Sensor (UDP na porta 8080)...")

	for {
		buffer := make([]byte, 1024) // Suficiente para JSON de sensores
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			// Ignora erros de leitura (ex: buffer curto) e continua.
			continue
		}
		// Encaminha o payload exato para os dois pipelines.
		chCliente <- buffer[:n]
		DadosSensor <- buffer[:n]
	}
}

// ==========================================
// RECEÇÃO DE COMANDOS DO CLIENTE (TCP) COM PROTEÇÃO DE FALHAS
// ==========================================
// recebecliente implementa o servidor TCP que aceita comandos manuais vindos
// da aplicação cliente (modo COMANDO).
// Cada conexão é tratada de forma síncrona e fechada após resposta.
// Inclui verificação de disponibilidade do atuador alvo antes de encaminhar o comando,
// fornecendo feedback imediato ao utilizador.
func recebecliente(chAtuador chan<- ComandoAtuador) {
	port := os.Getenv("CLIENTE_PORT")
	if port == "" {
		port = "8080" // Deve ser diferente da porta UDP para sensores; aqui usa-se a mesma mas protocolos distintos (TCP vs UDP).
	}

	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Printf("Erro no Listen TCP para cliente: %v\n", err)
		return
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}

		// Define um deadline de leitura para evitar que clientes maliciosos ou com problemas
		// mantenham uma conexão aberta indefinidamente sem enviar dados.
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))

		buffer := make([]byte, 1024)
		n, err := conn.Read(buffer)

		if err == nil {
			comando := strings.ToUpper(strings.TrimSpace(string(buffer[:n])))
			fmt.Printf("Comando manual do cliente recebido: %s\n", comando)

			// Mapeamento do comando textual para o alvo lógico correspondente.
			alvo := "DESCONHECIDO"
			if comando == "ABRIR" || comando == "FECHAR" {
				alvo = "BARREIRA"
			} else if comando == "LIGAR_ALARME" || comando == "DESLIGAR_ALARME" {
				alvo = "ALARME"
			}

			// Verifica se existe pelo menos uma conexão ativa para o alvo determinado.
			muAtuadores.Lock()
			atuadorOnline := false
			for _, tipo := range atuadoresAtivos {
				if tipo == alvo {
					atuadorOnline = true
					break
				}
			}
			muAtuadores.Unlock()

			// Prepara a resposta síncrona para o cliente que enviou o comando.
			conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
			if !atuadorOnline && alvo != "DESCONHECIDO" {
				msgErro := fmt.Sprintf("FALHA: Nao e possivel executar '%s'. O atuador %s esta DESCONECTADO.\n", comando, alvo)
				conn.Write([]byte(msgErro))
			} else if alvo != "DESCONHECIDO" {
				// Atualiza o estado interno mantido em memória para refletir a intenção do comando.
				mu.Lock()
				if alvo == "BARREIRA" {
					barreiraAberta = (comando == "ABRIR")
				} else if alvo == "ALARME" {
					alarmeLigado = (comando == "LIGAR_ALARME")
				}
				mu.Unlock()

				// Encaminha o comando para o broker, que o entregará ao atuador correto.
				chAtuador <- ComandoAtuador{Alvo: alvo, Acao: comando}
				conn.Write([]byte("SUCESSO: Comando aceite e roteado para " + alvo + "\n"))
			} else {
				conn.Write([]byte("ERRO: Comando nao reconhecido.\n"))
			}
		}

		// Independentemente do resultado, a conexão é encerrada.
		conn.Close()
	}
}

// enviacliente reencaminha os dados brutos dos sensores (via UDP) para o cliente monitor.
// O endereço do monitor é obtido da variável de ambiente CLIENTE_ADDR.
// A conexão UDP é estabelecida uma única vez e reutilizada para todas as mensagens.
func enviacliente(ch <-chan []byte) {
	clienteAddr := os.Getenv("CLIENTE_ADDR")
	if clienteAddr == "" {
		clienteAddr = "192.168.0.118:8081" // Endereço padrão do monitor (porta UDP)
	}
	conn, _ := net.Dial("udp", clienteAddr)
	for {
		msg := <-ch
		if conn != nil {
			conn.Write(msg) // Envia o payload exato recebido do sensor.
		}
	}
}
