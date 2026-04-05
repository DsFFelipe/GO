package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
)

var (
	barreiraFechada bool       // Estado da barreira
	mu              sync.Mutex // Garante acesso seguro entre UDP e TCP
)

func main() {
	fmt.Println("ATUADOR 1 (Barreira) - Iniciado")
	fmt.Println("Regra: Se > 50, fecha. Só abre via comando manual do Cliente.")

	go recebesensores() // Escuta telemetria (UDP)
	go recebecliente()  // Escuta comandos (TCP)

	select {}
}

func recebesensores() {
	addr, _ := net.ResolveUDPAddr("udp", ":8080")
	conn, _ := net.ListenUDP("udp", addr)
	defer conn.Close()

	for {
		buffer := make([]byte, 1024)
		n, _, _ := conn.ReadFromUDP(buffer)
		msg := strings.TrimSpace(string(buffer[:n]))

		// Converte valor do sensor
		valor, err := strconv.Atoi(msg)
		if err != nil {
			continue
		}

		mu.Lock()
		// Lógica: Se o sensor detectar perigo (>50), fecha a barreira
		if valor > 50 && !barreiraFechada {
			barreiraFechada = true
			fmt.Printf("[ALERTA] Sensor em %d: FECHANDO BARREIRA POR SEGURANÇA.\n", valor)
		} else if valor <= 50 {
			// Se o valor baixar, ela NÃO abre sozinha conforme solicitado
			fmt.Printf("[MONITORAMENTO] Valor: %d | Barreira continua: %s\n", valor, statusBarreira())
		}
		mu.Unlock()
	}
}

func recebecliente() {
	ln, _ := net.Listen("tcp", ":8080")
	defer ln.Close()

	for {
		conn, _ := ln.Accept()
		buffer := make([]byte, 1024)
		n, _ := conn.Read(buffer)
		comando := strings.ToUpper(strings.TrimSpace(string(buffer[:n])))

		mu.Lock()
		if comando == "ABRIR" || comando == "OPEN" {
			if barreiraFechada {
				barreiraFechada = false
				fmt.Println("[COMANDO CLIENTE] Barreira REABERTA manualmente.")
				conn.Write([]byte("Sucesso: Barreira aberta.\n"))
			} else {
				conn.Write([]byte("Aviso: Barreira já estava aberta.\n"))
			}
		} else {
			conn.Write([]byte("Erro: Comando não reconhecido.\n"))
		}
		mu.Unlock()
		conn.Close()
	}
}

// Função auxiliar para logs
func statusBarreira() string {
	if barreiraFechada {
		return "FECHADA"
	}
	return "ABERTA"
}
