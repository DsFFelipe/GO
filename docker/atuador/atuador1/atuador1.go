package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

// Variáveis de estado global.
var (
	// barreiraAberta representa o estado físico simulado do hardware.
	barreiraAberta bool = true
	// mu é um Mutex (Mutual Exclusion) necessário para evitar condições de corrida (race conditions),
	// garantindo que apenas uma goroutine altere ou leia o estado por vez.
	mu sync.Mutex
)

func main() {
	// Define o endereço do servidor. Prioriza variáveis de ambiente para facilitar
	// o uso de containers (Docker) ou ambientes de produção.
	servidorAddr := os.Getenv("SERVIDOR_ADDR")
	if servidorAddr == "" {
		servidorAddr = "192.168.0.71:8082"
	}

	fmt.Printf("Atuador conectando ao Servidor em %s...\n", servidorAddr)

	// Estabelece uma conexão TCP.
	// ordem dos comandos, é crítica para o controle de hardware.
	conn, err := net.Dial("tcp", servidorAddr)
	if err != nil {
		fmt.Printf("Falha ao conectar: %v\n", err)
		return
	}
	defer conn.Close() // Garante o fechamento do socket ao encerrar a execução.

	// Protocolo de Registro: O servidor precisa saber quem se conectou.
	// O '\n' é usado como delimitador de mensagem no stream TCP (framing).
	conn.Write([]byte("REGISTRO:BARREIRA\n"))
	fmt.Println("Registrado! Aguardando comandos...")

	// Envia o estado inicial (Open/Closed) para sincronizar o servidor com o estado atual do atuador.
	enviarEstado(conn)

	// Buffer para armazenar os dados recebidos do socket.
	buffer := make([]byte, 1024)
	for {
		// Leitura bloqueante: o loop aguarda até que o servidor envie algo.
		n, err := conn.Read(buffer)
		if err != nil {
			fmt.Println("Conexão com o servidor perdida.")
			break
		}

		// Converte os bytes em string, remove espaços/quebras de linha e padroniza para maiúsculas.
		comando := strings.ToUpper(strings.TrimSpace(string(buffer[:n])))
		executarComando(comando, conn)
	}
}

// executarComando processa a lógica de negócio baseada na mensagem recebida.
func executarComando(comando string, conn net.Conn) {
	// Seção Crítica: Protege o acesso à variável barreiraAberta.
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
	mu.Unlock() // Libera o recurso para outros processos.

	// Feedback: Após atuar, o dispositivo confirma a mudança de estado para o servidor.
	enviarEstado(conn)
}

// enviarEstado serializa o estado atual em JSON e o transmite via rede.
func enviarEstado(conn net.Conn) {
	mu.Lock()
	estado := barreiraAberta
	mu.Unlock()

	// Estrutura de dados para o protocolo de comunicação (DTo - Data Transfer Object).
	dados := map[string]interface{}{
		"id":     "BARREIRA",
		"ligado": estado,
	}

	// Transforma o mapa em uma string JSON formatada em bytes.
	b, _ := json.Marshal(dados)

	// Escrita no socket. Adicionamos '\n' para que o servidor saiba onde termina o objeto JSON.
	conn.Write(append(b, '\n'))
}
