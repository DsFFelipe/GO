package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
)

type Dados struct {
	Tipo       string `json:"tipo"`
	Valor      int    `json:"valor"`
	Localidade string `json:"localidade"`
}

// VARIÁVEIS GLOBAIS
var (
	// Estado lógico da decisão automática
	barreiraAberta bool       = true
	mu             sync.Mutex 

	// Pool de conexões dinâmicas para os atuadores
	atuadoresAtivos = make(map[net.Conn]bool)
	muAtuadores     sync.Mutex
)

func main() {
	sensorParaClienteChan := make(chan []byte)
	DadosSensor := make(chan []byte)
	clienteChan := make(chan []byte)

	// Inicia os fluxos originais
	go recebecliente(clienteChan)
	go recebesensor(sensorParaClienteChan, DadosSensor)
	go enviacliente(sensorParaClienteChan)
	go processarDecisao(DadosSensor, clienteChan)

	// INICIA A NOVA ARQUITETURA DE ATUADORES DINÂMICOS
	go recebeAtuadores()
	go enviaComandosParaAtuadores(clienteChan)

	select {} // Mantém o servidor vivo
}

// ==========================================
// NOVA LÓGICA: BROKER DE ATUADORES (TCP: 8082)
// ==========================================
func recebeAtuadores() {
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
		muAtuadores.Lock()
		atuadoresAtivos[conn] = true
		muAtuadores.Unlock()
		fmt.Printf("Novo atuador conectado: %s\n", conn.RemoteAddr().String())
	}
}

func enviaComandosParaAtuadores(ch <-chan []byte) {
	for {
		msg := <-ch

		muAtuadores.Lock()
		for conn := range atuadoresAtivos {
			_, err := conn.Write(msg)
			if err != nil {
				fmt.Printf("Atuador desconectado: %s\n", conn.RemoteAddr().String())
				conn.Close()
				delete(atuadoresAtivos, conn)
			}
		}
		muAtuadores.Unlock()
	}
}

// ==========================================
// LÓGICA ORIGINAL DE SENSORES E CLIENTES
// ==========================================
func recebesensor(chCliente chan<- []byte, DadosSensor chan<- []byte) {
	endr, err := net.ResolveUDPAddr("udp", "0.0.0.0:8080")
	if err != nil {
		return
	}
	conn, err := net.ListenUDP("udp", endr)
	if err != nil {
		return
	}
	defer conn.Close()

	fmt.Println("Servidor aguardando dados do Sensor (UDP na porta 8080)...")

	for {
		buffer := make([]byte, 1024)
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			continue
		}
		chCliente <- buffer[:n]
		DadosSensor <- buffer[:n]
	}
}

func recebecliente(ch chan<- []byte) {
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
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

func enviacliente(ch <-chan []byte) {
	clienteAddr := os.Getenv("CLIENTE_ADDR")
	if clienteAddr == "" {
		clienteAddr = "192.168.0.118:8080"
	}

	conn, err := net.Dial("udp", clienteAddr)
	if err != nil {
		return
	}
	for {
		msg := <-ch
		conn.Write(msg)
	}
}

func processarDecisao(chSensor <-chan []byte, chAtuador chan<- []byte) {
	for {
		rawBytes := <-chSensor

		var d Dados
		err := json.Unmarshal(rawBytes, &d)
		if err != nil {
			continue
		}

		mu.Lock()
		fmt.Printf("\n[TELEMETRIA] %s em %s: %d\n", d.Tipo, d.Localidade, d.Valor)

		if d.Valor > 70 && barreiraAberta {
			barreiraAberta = false
			fmt.Println("ALERTA: Nível crítico! Fechando barreira automaticamente.")
			chAtuador <- []byte("FECHAR")
		} else if d.Valor < 50 && !barreiraAberta {
			barreiraAberta = true
			fmt.Println("STATUS: Nível seguro. Abrindo barreira automaticamente.")
			chAtuador <- []byte("ABRIR")
		}
		mu.Unlock()
	}
}