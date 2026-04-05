package main

import (
	"fmt"
	"net"
	"strings"
	"sync"
)

// Estado global da barreira e controle de concorrência
var (
	barreiraAberta bool       = true
	mu             sync.Mutex // Garante que apenas uma rotina altere o estado por vez
)

func main() {
	// Inicia a escuta de comandos via TCP em uma goroutine
	go recebeComandosTCP()

	// Impede que a função main termine, mantendo o container ativo
	select {}
}

func recebeComandosTCP() {
	// O servidor TCP escuta na porta 8080
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Printf("Erro ao abrir socket TCP: %v\n", err)
		return
	}
	defer ln.Close()

	fmt.Println("Atuador aguardando comandos via TCP...")

	for {
		// Aceita novas conexões vindas do servidor
		conn, err := ln.Accept()
		if err != nil {
			fmt.Printf("Erro ao aceitar conexão TCP: %v\n", err)
			continue
		}

		// Processa cada conexão em uma goroutine separada
		go handleTCPConnection(conn)
	}
}

func handleTCPConnection(conn net.Conn) {
	defer conn.Close()

	// Buffer para leitura dos dados enviados pelo servidor
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		return
	}

	// Converte bytes para string, remove espaços e coloca em maiúsculas
	comando := strings.ToUpper(strings.TrimSpace(string(buffer[:n])))

	mu.Lock()
	if comando == "ABRIR" {
		barreiraAberta = true
		fmt.Println("[SERVIDOR] Comando recebido: Abrindo barreira.")
	} else if comando == "FECHAR" {
		barreiraAberta = false
		fmt.Println("[SERVIDOR] Comando recebido: Fechando barreira.")
	}

	fmt.Printf("Estado atual da barreira: Aberta = %v\n", barreiraAberta)
	mu.Unlock()
}
