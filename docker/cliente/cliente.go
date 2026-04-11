package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// ====================================================
// DEFINIÇÃO DAS ESTRUTURAS DE DADOS (DTOs de Rede)
// ====================================================

// MensagemSensor define o formato dos dados transmitidos via UDP pelos sensores IoT.
type MensagemSensor struct {
	Tipo       string `json:"tipo"`       // Ex: "TEMPERATURA", "PRESENCA"
	Valor      int    `json:"valor"`      // Leitura analógica/digital
	Localidade string `json:"localidade"` // Identificador geográfico lógico (ex: "Entrada")
}

// EstadoAtuador é o DTo enviado pelos atuadores via TCP.
// A estrutura é idêntica à enviada por 'atuador1' para garantir a desserialização correta.
type EstadoAtuador struct {
	ID     string `json:"id"`     // Ex: "BARREIRA"
	Ligado bool   `json:"ligado"` // true = aberta/ligada, false = fechada/desligada
}

// SensorStatus é uma estrutura utilizada internamente pelo monitor.
// Ela acopla o último dado recebido com um timestamp para implementar o
// mecanismo de Heartbeat/Timeout (Critério 8). Sem esta informação temporal,
// o sistema não teria como distinguir um sensor silencioso de um sensor morto.
type SensorStatus struct {
	Dados       MensagemSensor
	UltimoVisto time.Time // Momento exato da última atualização (base para timeout)
}

// ====================================================
// VARIÁVEIS DE ESTADO GLOBAL E SINCRONIZAÇÃO
// ====================================================

var (
	// sensoresAtivos mantém o estado volátil de todos os sensores observados.
	// A chave é composta por Tipo-Localidade para evitar colisões entre sensores distintos.
	// Acesso concorrente: goroutine UDP (leitura de rede) e goroutine de Render (leitura para UI).
	sensoresAtivos = make(map[string]SensorStatus)
	mu             sync.Mutex // Mutex para proteger o mapa sensoresAtivos (Race Condition Mitigation)

	// estadosAtuadores armazena o último estado confirmado recebido via TCP.
	// Acesso concorrente: goroutine TCP Listener (escrita) e Render Loop (leitura).
	estadosAtuadores = make(map[string]EstadoAtuador)
	muAtuadores      sync.Mutex
)

// ====================================================
// Main
// ====================================================
func main() {
	// O binário é único, mas o comportamento é definido por variável de ambiente.
	// Padrão 12-Factor App: permite deploy idêntico para funções diferentes (Monitor vs Terminal).
	modo := os.Getenv("CLIENTE_MODO")
	if modo == "" {
		modo = "MONITOR" // Fallback seguro se a variável não for setada.
	}

	if modo == "MONITOR" {
		// Arquitetura Concorrente: O monitor precisa escutar dois protocolos diferentes simultaneamente.
		// UDP (8083) para sensores
		// TCP (8084) para atuadores
		go recebe()             // Goroutine bloqueante de leitura UDP.
		go recebeAtuadoresTCP() // Goroutine bloqueante de escuta TCP.

		// O loop de renderização roda na thread principal (main goroutine).
		renderLoop()
	} else if modo == "COMANDO" {
		// Modo Terminal: Usa um canal para comunicar a goroutine de I/O (stdin) com a de rede.
		// O buffer do canal é 0 para sincronização estrita, mas não é necessário buffer grande.
		comandoChan := make(chan string)
		go envia(comandoChan)        // Gerencia sockets e timeouts de rede.
		capturarEntrada(comandoChan) // Bloqueia lendo stdin.
	} else {
		fmt.Println("Erro: CLIENTE_MODO deve ser 'MONITOR' ou 'COMANDO'")
		os.Exit(1)
	}
}

// ====================================================
// MODO: MONITOR - RECEPTOR UDP (SENSORES)
// ====================================================
// recebe inicia um servidor UDP na porta definida (padrão 8083).

func recebe() {
	port := os.Getenv("MONITOR_UDP_PORT")
	if port == "" {
		port = "8083"
	}

	endr, err := net.ResolveUDPAddr("udp", ":"+port)
	if err != nil {
		fmt.Printf("Falha na resolução de endereço UDP: %v\n", err)
		return
	}
	conn, err := net.ListenUDP("udp", endr)
	if err != nil {
		fmt.Printf("Falha ao abrir socket UDP: %v\n", err)
		return
	}
	defer conn.Close()

	fmt.Printf("Monitor UDP escutando em 0.0.0.0:%s\n", port)

	for {
		buffer := make([]byte, 1024) // Tamanho suficiente para um JSON pequeno (< 1KB)
		n, _, err := conn.ReadFromUDP(buffer)
		if err == nil {
			var dados MensagemSensor
			if err := json.Unmarshal(buffer[:n], &dados); err == nil {
				mu.Lock()
				chave := fmt.Sprintf("%s-%s", dados.Tipo, dados.Localidade)
				// Regista o tempo exato em que o pacote foi recebido
				sensoresAtivos[chave] = SensorStatus{
					Dados:       dados,
					UltimoVisto: time.Now(),
				}
				mu.Unlock()
			}
		}
	}
}

// ====================================================
// MODO: MONITOR - RECEPTOR TCP (ATUADORES)
// ====================================================
// recebeAtuadoresTCP implementa um servidor TCP simples para receber confirmações
// de estado dos atuadores (ex: "BARREIRA").
func recebeAtuadoresTCP() {
	port := os.Getenv("MONITOR_TCP_PORT")
	if port == "" {
		port = "8084"
	}

	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Printf("Erro ao iniciar ouvinte TCP para atuadores: %v\n", err)
		return
	}
	defer ln.Close()
	fmt.Printf("Monitor TCP escutando em 0.0.0.0:%s\n", port)

	for {
		conn, err := ln.Accept()
		if err != nil {
			// Pode ocorrer erro temporário, continuamos aceitando novas conexões.
			continue
		}

		// Processamento síncrono dentro da goroutine principal do listener.
		// Para alta concorrência, o ideal seria `go handleConnection(conn)`, mas
		// dado que atuadores enviam dados apenas esporadicamente, a abordagem simples é suficiente.
		buffer := make([]byte, 1024)
		n, err := conn.Read(buffer)
		if err == nil {
			var est EstadoAtuador
			if err := json.Unmarshal(buffer[:n], &est); err == nil {
				muAtuadores.Lock()
				estadosAtuadores[est.ID] = est // Sobrescreve ou adiciona o estado confirmado.
				muAtuadores.Unlock()
			}
		}
		// Encerra a conexão imediatamente. O atuador não espera resposta em nível de aplicação.
		conn.Close()
	}
}

// ====================================================
// MODO: MONITOR - INTERFACE DE USUÁRIO (REFRESH LOOP)
// ====================================================
// renderLoop é utiliza um Ticker do pacote time para
// garantir uma taxa de atualização fixa (200ms) independente do tempo de processamento.
// Isso evita "flood" de escrita no terminal e mantém a carga da CPU baixa.
func renderLoop() {
	// Ticker: mecanismo idiomático em Go para loops periódicos.
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	// Timeout para considerar um sensor como inativo (falha de comunicação).
	// Valor escolhido empiricamente: 10 segundos sem pacotes UDP é tempo suficiente
	// para assumir que o sensor desligou ou a rede falhou.
	timeoutSensor := 10 * time.Second

	for range ticker.C {
		// Limpeza do terminal para efeito de "Dashboard em Tempo Real".
		// O comando 'clear' é específico para sistemas Unix/Linux (funciona em Alpine/Debian).
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		_ = cmd.Run() // Ignoraerro caso o SO não suporte (fallback visual com scroll)

		fmt.Println("=== MONITOR DE TELEMETRIA EM TEMPO REAL ===")

		// --- Renderização da Tabela de Sensores ---
		mu.Lock()
		if len(sensoresAtivos) == 0 {
			fmt.Println("Aguardando pacotes de sensores na rede...")
		} else {
			// Itera sobre o mapa (ordem não determinística, aceitável para UI simples)
			for _, status := range sensoresAtivos {
				// Lógica de Detecção de Falha (Heartbeat):
				// Se a diferença entre Agora e o ÚltimoVisto exceder o timeout, declaramos falha.
				if time.Since(status.UltimoVisto) > timeoutSensor {
					fmt.Printf("[ALERTA] Sensor: %-20s | Local: %-10s | STATUS: DESCONECTADO (Timeout)\n",
						status.Dados.Tipo, status.Dados.Localidade)
				} else {
					fmt.Printf("Sensor: %-20s | Local: %-10s | Valor: %d\n",
						status.Dados.Tipo, status.Dados.Localidade, status.Dados.Valor)
				}
			}
		}
		mu.Unlock()

		// --- Renderização da Tabela de Atuadores ---
		fmt.Println("\n--- ESTADO DOS ATUADORES (TCP) ---")
		muAtuadores.Lock()
		if len(estadosAtuadores) == 0 {
			fmt.Println("Aguardando sincronização de atuadores...")
		} else {
			for _, est := range estadosAtuadores {
				// Mapeamento semântico: Ligado = true -> Aberta
				status := "FECHADA / DESLIGADO"
				if est.Ligado {
					status = "ABERTA / LIGADO"
				}
				fmt.Printf("Atuador: %-10s | Estado Confirmado: %s\n", est.ID, status)
			}
		}
		muAtuadores.Unlock()
		fmt.Println("===========================================")
	}
}

// ====================================================
// MODO: COMANDO - ENTRADA DO UTILIZADOR (STDIN)
// ====================================================
// capturarEntrada lê comandos do teclado e os injeta no canal de comunicação.
// O uso de bufio.Scanner permite leitura linha-a-linha bloqueante, libertando a CPU.
func capturarEntrada(ch chan<- string) {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Terminal de Comandos Ativo.")
	fmt.Println("Instruções válidas: ABRIR, FECHAR, LIGAR_ALARME, DESLIGAR_ALARME")
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			// EOF (Ctrl+D) ou erro de leitura. Encerra o loop.
			break
		}
		msg := strings.TrimSpace(scanner.Text())
		if msg != "" {
			// Envia para a goroutine de rede. Se o canal estiver cheio, bloqueia.
			// Como não há buffer, isso age como backpressure natural.
			ch <- msg
		}
	}
	close(ch) // Sinaliza para a goroutine 'envia' que não haverá mais comandos.
}

// ====================================================
// MODO: COMANDO - CLIENTE DE REDE (ENVIO DE COMANDOS)
// ====================================================
// envia consome comandos do canal e os transmite ao Servidor Central.
// Implementa mecanismos de Timeout para evitar que o terminal
// fique bloqueado eternamente caso o servidor esteja offline ou lento.
func envia(ch <-chan string) {
	servidorAddr := os.Getenv("SERVIDOR_ADDR")
	if servidorAddr == "" {
		servidorAddr = "servidor:8080" // Nome DNS padrão para ambientes Docker
	}

	for msg := range ch {
		// TIMEOUT DE CONEXÃO (DialTimeout):
		// Se o servidor não responder ao SYN em 3 segundos, falha rápido.
		// Evita que o usuário fique esperando 30 segundos ou mais pelo timeout padrão do SO.
		conn, err := net.DialTimeout("tcp", servidorAddr, 3*time.Second)
		if err != nil {
			fmt.Printf("\n[ERRO CRÍTICO] Servidor inacessível: %v\n> ", err)
			continue
		}

		// TIMEOUT DE ESCRITA (SetWriteDeadline):
		// Garante que a operação conn.Write não bloqueie infinitamente se os buffers do kernel estiverem cheios.
		conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
		_, err = conn.Write([]byte(msg))
		if err != nil {
			fmt.Printf("\n[ERRO] Falha ao enviar comando para o socket: %v\n> ", err)
			conn.Close()
			continue
		}

		// TIMEOUT DE LEITURA (SetReadDeadline):
		// Aguarda a confirmação do servidor por no máximo 3 segundos.
		// Isso é crítico para a experiência do usuário: ele sabe que o comando foi entregue ou não.
		conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		respBuffer := make([]byte, 1024)
		n, err := conn.Read(respBuffer)
		if err != nil {
			// Se ocorrer timeout ou conexão resetada, informa o usuário.
			fmt.Printf("\n[AVISO] Servidor não confirmou o comando (Timeout de Leitura)\n> ")
		} else {
			// Exibe a resposta do servidor (Ex: "Comando ABRIR enviado ao atuador")
			fmt.Printf("\n[RESPOSTA DO SERVIDOR] %s> ", string(respBuffer[:n]))
		}

		// Encerra a conexão corretamente. O protocolo é "Request/Response" simples,
		// não mantem conexão persistente para evitar vazamento de file descriptors.
		conn.Close()
	}
}
