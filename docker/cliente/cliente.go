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

// Nova estrutura para o Critério 8: Controlo de Heartbeat/Timeout
type SensorStatus struct {
	Dados       MensagemSensor
	UltimoVisto time.Time
}

// Variáveis de estado global e os seus respetivos Mutexes
var (
	sensoresAtivos = make(map[string]SensorStatus)
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

	for {
		buffer := make([]byte, 1024)
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
	timeoutSensor := 10 * time.Second // Limite de inatividade para considerar falha

	for range ticker.C {
		// Executa o comando nativo do Linux (Alpine) para limpar o terminal
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		cmd.Run()

		fmt.Println("=== MONITOR DE TELEMETRIA EM TEMPO REAL ===")

		// Renderiza os Sensores
		mu.Lock()
		if len(sensoresAtivos) == 0 {
			fmt.Println("Aguardando pacotes de sensores na rede...")
		} else {
			for _, status := range sensoresAtivos {
				// Lógica de falha: Se o tempo desde o último pacote for maior que o timeout
				if time.Since(status.UltimoVisto) > timeoutSensor {
					fmt.Printf("[ALERTA] Sensor: %-20s | Local: %-10s | STATUS: DESCONECTADO (Timeout)\n", status.Dados.Tipo, status.Dados.Localidade)
				} else {
					fmt.Printf("Sensor: %-20s | Local: %-10s | Valor: %d\n", status.Dados.Tipo, status.Dados.Localidade, status.Dados.Valor)
				}
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
		// USO DE TIMEOUT NO DIAL: Evita que o cliente bloqueie se o IP do servidor não existir
		conn, err := net.DialTimeout("tcp", servidorAddr, 3*time.Second)
		if err != nil {
			fmt.Printf("\n[ERRO CRÍTICO] Servidor inacessível: %v\n> ", err)
			continue
		}

		// USO DE TIMEOUT DE ESCRITA: Protege contra buffers de rede cheios
		conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
		_, err = conn.Write([]byte(msg))
		if err != nil {
			fmt.Printf("\n[ERRO] Falha ao enviar comando para o socket: %v\n> ", err)
			conn.Close()
			continue
		}

		// USO DE TIMEOUT DE LEITURA: Aguarda o feedback garantido do servidor
		conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		respBuffer := make([]byte, 1024)
		n, err := conn.Read(respBuffer)
		if err != nil {
			fmt.Printf("\n[AVISO] Servidor não confirmou o comando (Timeout de Leitura)\n> ")
		} else {
			// Imprime a resposta exata do servidor (Sucesso ou Falha do Atuador)
			fmt.Printf("\n[RESPOSTA DO SERVIDOR] %s> ", string(respBuffer[:n]))
		}

		conn.Close()
	}
}
