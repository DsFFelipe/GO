package main

import (
	"encoding/json" // Necessário para converter bytes em estruturas de dados
	"fmt"
	"net"
	"os" // Pacote adicionado para acessar variáveis de ambiente
	"sync"
)

// Estrutura para interpretar os dados vindos do sensor
type Dados struct {
	Tipo       string `json:"tipo"`
	Valor      int    `json:"valor"`
	Localidade string `json:"localidade"`
}

var (
	// Pool de conexões TCP ativas com atuadores
	atuadoresAtivos = make(map[net.Conn]bool)
	muAtuadores     sync.Mutex 
)

func main() {
	// Canais para comunicação entre goroutines
	sensorParaClienteChan := make(chan []byte)
	DadosSensor := make(chan []byte)
	clienteChan := make(chan []byte)

	// Inicia os fluxos de recebimento e envio
	go recebecliente(clienteChan)
	// Ajustado para usar o canal DadosSensor declarado acima
	go recebesensor(sensorParaClienteChan, DadosSensor)

	go enviacliente(sensorParaClienteChan)

	// Inicia a lógica de decisão automática
	go processarDecisao(DadosSensor, clienteChan)

	go enviaatuadorTCP(clienteChan)

	select {} // Mantém o servidor vivo
}

func recebesensor(chCliente chan<- []byte, DadosSensor chan<- []byte) {
	endr, err := net.ResolveUDPAddr("udp", "0.0.0.0:8080")
	if err != nil {
		fmt.Printf("Erro no endereço UDP: %v\n", err)
		return
	}
	conn, err := net.ListenUDP("udp", endr)
	if err != nil {
		fmt.Printf("Erro ao abrir porta UDP: %v\n", err)
		return
	}
	defer conn.Close()

	fmt.Println("Servidor aguardando dados do Sensor (UDP)...")

	for {
		buffer := make([]byte, 1024)
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Printf("Erro na leitura UDP: %v\n", err)
			continue
		}

		// Repassa os bytes brutos para os canais de envio
		chCliente <- buffer[:n]
		DadosSensor <- buffer[:n]
	}
}

func recebecliente(ch chan<- []byte) {
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Printf("Erro no Listen TCP: %v\n", err)
		return
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		buffer := make([]byte, 1024)
		n, _ := conn.Read(buffer)
		ch <- buffer[:n]
		fmt.Printf("Comando do cliente recebido: %s\n", string(buffer[:n]))
		conn.Close()
	}
}

func recebeAtuadores() {
	// Abre uma porta TCP exclusiva para os atuadores se conectarem
	ln, err := net.Listen("tcp", ":8082") 
	if err != nil {
		fmt.Printf("Erro no Listen TCP para atuadores: %v\n", err)
		return
	}
	defer ln.Close()

	fmt.Println("Servidor aguardando conexões de Atuadores (TCP na porta 8082)...")

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		
		// Quando um novo atuador se conecta, registra no pool
		muAtuadores.Lock()
		atuadoresAtivos[conn] = true
		muAtuadores.Unlock()
		
		fmt.Printf("Novo atuador conectado: %s\n", conn.RemoteAddr().String())
	}
}

func enviacliente(ch <-chan []byte) {
	// Busca o endereço do cliente nas variáveis de ambiente
	clienteAddr := os.Getenv("CLIENTE_ADDR")
	if clienteAddr == "" {
		clienteAddr = "cliente:8080" // Valor padrão de fallback
	}

	// Utiliza a variável resolvida no lugar de uma string fixa
	conn, err := net.Dial("udp", clienteAddr)
	if err != nil {
		return
	}
	for {
		msg := <-ch
		conn.Write(msg)
	}
}

// Lógica de processamento e decisão automática
func processarDecisao(chSensor <-chan []byte, chAtuador chan<- []byte) {
	for {
		// Recebe os bytes do canal do sensor
		rawBytes := <-chSensor

		var d Dados
		// Converte JSON em estrutura Go
		err := json.Unmarshal(rawBytes, &d)
		if err != nil {
			fmt.Println("Erro ao decodificar JSON:", err)
			continue
		}

		mu.Lock()
		fmt.Printf("\n[TELEMETRIA] %s em %s: %d\n", d.Tipo, d.Localidade, d.Valor)

		// Lógica automática: envia comandos de texto para o atuador via canal
		if d.Valor > 70 && barreiraAberta {
			barreiraAberta = false
			fmt.Println("ALERTA: Nível crítico! Fechando barreira automaticamente.")
			chAtuador <- []byte("FECHAR")
		} else if d.Valor < 50 && !barreiraAberta {
			barreiraAberta = true
			fmt.Println("STATUS: Nível seguro. Abrindo barreira automaticamente.")
			chAtuador <- []byte("ABRIR")
		}

		fmt.Printf("Estado da barreira: Aberta = %v\n", barreiraAberta)
		mu.Unlock()
	}
}

func enviaComandosParaAtuadores(ch <-chan []byte) {
	for {
		msg := <-ch
		
		muAtuadores.Lock()
		// Itera sobre todos os atuadores registrados
		for conn := range atuadoresAtivos {
			_, err := conn.Write(msg)
			if err != nil {
				// Se a escrita falhar, o atuador caiu ou foi desligado.
				// Remove a conexão inativa do pool para evitar vazamento de memória.
				fmt.Printf("Atuador desconectado: %s\n", conn.RemoteAddr().String())
				conn.Close()
				delete(atuadoresAtivos, conn)
			}
		}
		muAtuadores.Unlock()
	}
}