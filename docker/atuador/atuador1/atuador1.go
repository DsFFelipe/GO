package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

var (
	barreiraAberta bool = true
	mu             sync.Mutex
)

func main() {
	servidorAddr := os.Getenv("SERVIDOR_ADDR")
	if servidorAddr == "" {
		servidorAddr = "192.168.0.71:8082"
	}

	fmt.Printf("Atuador conectando ao Servidor em %s...\n", servidorAddr)

	conn, err := net.Dial("tcp", servidorAddr)
	if err != nil {
		fmt.Printf("Falha ao conectar: %v\n", err)
		return
	}
	defer conn.Close()

	// Adicionado o delimitador \n para a leitura em buffer do servidor
	conn.Write([]byte("REGISTRO:BARREIRA\n"))
	fmt.Println("Registrado! Aguardando comandos...")

	// Envia o estado inicial assim que conecta
	enviarEstado(conn)

	buffer := make([]byte, 1024)
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			fmt.Println("Conexão com o servidor perdida.")
			break
		}
		comando := strings.ToUpper(strings.TrimSpace(string(buffer[:n])))
		executarComando(comando, conn)
	}
}

func executarComando(comando string, conn net.Conn) {
	mu.Lock()
	if comando == "ABRIR" {
		barreiraAberta = true
	} else if comando == "FECHAR" {
		barreiraAberta = false
	} else {
		fmt.Printf("[AVISO] Comando desconhecido: %s\n", comando)
		mu.Unlock()
		return
	}
	fmt.Printf("Estado atual da barreira: Aberta = %v\n", barreiraAberta)
	mu.Unlock()

	// Retorna o feedback do estado atualizado para o servidor
	enviarEstado(conn)
}

func enviarEstado(conn net.Conn) {
	mu.Lock()
	estado := barreiraAberta
	mu.Unlock()

	dados := map[string]interface{}{
		"id":     "BARREIRA",
		"ligado": estado,
	}
	b, _ := json.Marshal(dados)
	conn.Write(append(b, '\n')) // Delimitador de stream TCP
}
