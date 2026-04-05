package main

import (
	"fmt"
	"net"
	"os" // Adicionado para acessar as variáveis de ambiente do sistema operacional
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
	// Busca a porta configurada no ambiente do sistema
	porta := os.Getenv("PORTA")

	// Define a porta 8080 como padrão caso nenhuma seja fornecida
	if porta == "" {
		porta = "8080"
	}

	// O servidor TCP escuta na porta dinâmica especificada
	// O formato exige dois pontos antes do número da porta (ex: ":8080")
	ln, err := net.Listen("tcp", ":"+porta)
	if err != nil {
		fmt.Printf("Erro ao abrir socket TCP na porta %s: %v\n", porta, err)
		return
	}
	defer ln.Close()

	fmt.Printf("Atuador aguardando comandos via TCP na porta %s...\n", porta)

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
