package main

import (
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

// Estado global da barreira e controle de concorrência
var (
	barreiraAberta bool       = true
	mu             sync.Mutex // Garante acesso seguro à variável de estado
)

func main() {
	// O atuador agora atua como cliente TCP e precisa do IP do Servidor
	servidorAddr := os.Getenv("SERVIDOR_ADDR")
	if servidorAddr == "" {
		servidorAddr = "192.168.0.71:8082" // IP do Servidor e porta dedicada a atuadores
	}

	fmt.Printf("Atuador conectando ao Servidor em %s...\n", servidorAddr)

	// Inicia a conexão com o servidor (O handshake TCP ocorre aqui)
	conn, err := net.Dial("tcp", servidorAddr)
	if err != nil {
		fmt.Printf("Falha ao conectar ao servidor: %v\n", err)
		return
	}
	defer conn.Close()

	fmt.Println("Conectado! Aguardando comandos do broker...")

	// Buffer para armazenar os bytes recebidos via rede
	buffer := make([]byte, 1024)
	
	// Loop infinito bloqueante. Ele pausa na linha conn.Read até que pacotes cheguem.
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			fmt.Println("Conexão com o servidor perdida (Broker desconectado).")
			break // Encerra o loop e o programa se a conexão cair
		}

		// Limpa os bytes recebidos e converte para string
		comando := strings.ToUpper(strings.TrimSpace(string(buffer[:n])))
		executarComando(comando) 
	}
}

// Função que isola a lógica de negócio estrutural do atuador
func executarComando(comando string) {
	// Trava o Mutex. Em arquiteturas concorrentes, isso impede condições de corrida 
	// ao ler e escrever na variável global barreiraAberta.
	mu.Lock()
	defer mu.Unlock() // Garante que o desbloqueio ocorra quando a função retornar

	if comando == "ABRIR" {
		if barreiraAberta {
			fmt.Println("[SERVIDOR] Comando recebido: ABRIR. A barreira já está aberta.")
		} else {
			barreiraAberta = true
			fmt.Println("[SERVIDOR] Comando recebido: Abrindo barreira.")
		}
	} else if comando == "FECHAR" {
		if !barreiraAberta {
			fmt.Println("[SERVIDOR] Comando recebido: FECHAR. A barreira já está fechada.")
		} else {
			barreiraAberta = false
			fmt.Println("[SERVIDOR] Comando recebido: Fechando barreira.")
		}
	} else {
		// Proteção contra payloads inesperados no socket TCP
		fmt.Printf("[AVISO] Comando desconhecido ignorado: %s\n", comando)
		return 
	}

	fmt.Printf("Estado atual da barreira: Aberta = %v\n", barreiraAberta)
}