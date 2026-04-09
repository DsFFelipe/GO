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

// Estruturas de dados
type MensagemSensor struct {
	Tipo       string `json:"tipo"`
	Valor      int    `json:"valor"`
	Localidade string `json:"localidade"`
}

type EstadoAtuador struct {
	ID     string `json:"id"`
	Ligado bool   `json:"ligado"`
}

// Variáveis de estado global e seus respectivos Mutexes
var (
	sensoresAtivos = make(map[string]MensagemSensor)
	mu             sync.Mutex

	estadosAtuadores = make(map[string]EstadoAtuador)
	muAtuadores      sync.Mutex
)

func main() {
	// Determina a responsabilidade do processo via variável de ambiente
	modo := os.Getenv("CLIENTE_MODO")
	if modo == "" {
		modo = "MONITOR" // Fallback
	}

	if modo == "MONITOR" {
		go recebe()             // Escuta Sensores via UDP (8083)
		go recebeAtuadoresTCP() // Escuta Atuadores via TCP (8084)
		renderLoop()            // Bloqueia a thread principal com a renderização
	} else if modo == "COMANDO" {
		comandoChan := make(chan string)
		go envia(comandoChan)
		capturarEntrada(comandoChan) // Bloqueia a thread principal esperando I/O
	} else {
		fmt.Println("Erro: CLIENTE_MODO deve ser 'MONITOR' ou 'COMANDO'")
		os.Exit(1)
	}
}

// ==========================================
// MODO: MONITOR
// ==========================================
func recebe() {
	endr, err := net.ResolveUDPAddr("udp", ":8083")
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

	for {
		buffer := make([]byte, 1024)
		n, _, err := conn.ReadFromUDP(buffer)
		if err == nil {
			var dados MensagemSensor
			if err := json.Unmarshal(buffer[:n], &dados); err == nil {
				mu.Lock()
				chave := fmt.Sprintf("%s-%s", dados.Tipo, dados.Localidade)
				sensoresAtivos[chave] = dados
				mu.Unlock()
			}
		}
	}
}

func recebeAtuadoresTCP() {
	ln, err := net.Listen("tcp", ":8084")
	if err != nil {
		fmt.Printf("Erro ao iniciar ouvinte TCP para atuadores: %v\n", err)
		return
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}

		buffer := make([]byte, 1024)
		n, err := conn.Read(buffer)
		if err == nil {
			var est EstadoAtuador
			if err := json.Unmarshal(buffer[:n], &est); err == nil {
				muAtuadores.Lock()
				estadosAtuadores[est.ID] = est
				muAtuadores.Unlock()
			}
		}
		conn.Close()
	}
}

func renderLoop() {
	// Utiliza um Ticker para garantir ciclos precisos a cada 500ms
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		// Sequência ANSI para limpar a tela e resetar o cursor
		fmt.Print("\033[H\033[2J")
		fmt.Println("=== MONITOR DE TELEMETRIA EM TEMPO REAL ===")

		// Renderiza os Sensores
		mu.Lock()
		if len(sensoresAtivos) == 0 {
			fmt.Println("Aguardando pacotes de sensores na rede...")
		} else {
			for _, dados := range sensoresAtivos {
				fmt.Printf("Sensor: %-20s | Local: %-10s | Valor: %d\n", dados.Tipo, dados.Localidade, dados.Valor)
			}
		}
		mu.Unlock()

		// Renderiza os Atuadores
		fmt.Println("\n--- ESTADO DOS ATUADORES (TCP) ---")
		muAtuadores.Lock()
		if len(estadosAtuadores) == 0 {
			fmt.Println("Aguardando sincronização de atuadores...")
		} else {
			for _, est := range estadosAtuadores {
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

// ==========================================
// MODO: COMANDO
// ==========================================
func capturarEntrada(ch chan<- string) {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Terminal de Comandos Ativo.")
	fmt.Println("Instruções válidas: ABRIR, FECHAR, LIGAR_ALARME, DESLIGAR_ALARME")
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		msg := strings.TrimSpace(scanner.Text())
		if msg != "" {
			ch <- msg
		}
	}
}

func envia(ch <-chan string) {
	servidorAddr := os.Getenv("SERVIDOR_ADDR")
	if servidorAddr == "" {
		servidorAddr = "servidor:8080"
	}

	for msg := range ch {
		conn, err := net.Dial("tcp", servidorAddr)
		if err != nil {
			fmt.Printf("\nErro de roteamento TCP ao servidor: %v\n", err)
			continue
		}
		conn.Write([]byte(msg))
		conn.Close()
	}
}
