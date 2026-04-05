package main

import (
	"fmt"
	"net"
	"strings"
	"sync"
)

// Estrutura para espelhar o JSON do sensor
type DadosSensor struct {
	Tipo       string `json:"tipo"`
	Valor      int    `json:"valor"`
	Localidade string `json:"localidade"`
}

// Estado global da barreira e controle de concorrência
var (
	barreiraAberta bool       = true
	mu             sync.Mutex // Garante que apenas uma rotina altere o estado por vez
)

func main() {
	// Inicia a escuta de comandos do cliente em uma rotina separada
	go recebeComandosTCP()

	// Mantém a escuta de telemetria na rotina principal
}

func recebeComandosTCP() {
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Printf("Erro ao abrir socket TCP: %v\n", err)
		return
	}
	defer ln.Close()

	fmt.Println("Atuador aguardando comandos via TCP...")

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Printf("Erro ao aceitar conexão TCP: %v\n", err)
			continue
		}

		go handleTCPConnection(conn)
	}
}

func handleTCPConnection(conn net.Conn) {
	defer conn.Close()
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		return
	}

	comando := strings.ToUpper(strings.TrimSpace(string(buffer[:n])))

	mu.Lock()
	if comando == "ABRIR" {
		barreiraAberta = true
		fmt.Println("[CLIENTE] Comando recebido: Abrindo barreira.")
	} else if comando == "FECHAR" {
		barreiraAberta = false
		fmt.Println("[CLIENTE] Comando recebido: Fechando barreira.")
	}
	fmt.Printf("Estado atual da barreira: Aberta = %v\n", barreiraAberta)
	mu.Unlock()
}

/*func recebeDadosUDP() {
	addr, err := net.ResolveUDPAddr("udp", ":8080")
	if err != nil {
		fmt.Printf("Erro ao resolver endereço: %v\n", err)
		return
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Printf("Erro ao abrir socket UDP: %v\n", err)
		return
	}
	defer conn.Close()

	fmt.Println("Atuador aguardando dados via UDP...")

	for {
		buffer := make([]byte, 1024)
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Printf("Erro na leitura: %v\n", err)
			continue
		}

		var dados DadosSensor
		err = json.Unmarshal(buffer[:n], &dados)
		if err != nil {
			fmt.Printf("Erro ao decodificar JSON: %v\n", err)
			continue
		}

		processarDecisao(dados)
	}
}

func processarDecisao(d DadosSensor) {
	mu.Lock()
	defer mu.Unlock()

	fmt.Printf("\n[TELEMETRIA] %s em %s: %d\n", d.Tipo, d.Localidade, d.Valor)

	// Lógica automática baseada nos limiares solicitados
	if d.Valor > 70 && barreiraAberta {
		barreiraAberta = false
		fmt.Println("ALERTA: Nível crítico! Fechando barreira automaticamente.")
	} else if d.Valor < 50 && !barreiraAberta {
		barreiraAberta = true
		fmt.Println("STATUS: Nível seguro. Abrindo barreira automaticamente.")
	}

	fmt.Printf("Estado da barreira: Aberta = %v\n", barreiraAberta)
}*/
