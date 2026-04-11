package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strconv"
	"sync/atomic"
	"time"
)

func main() {
	servidorIP := os.Getenv("SERVER_IP")
	if servidorIP == "" {
		servidorIP = "192.168.15.187"
	}

	// Configuração de Sensores (UDP)
	numSensoresStr := os.Getenv("NUM_SENSORES")
	numSensores := 1000
	if numSensoresStr != "" {
		numSensores, _ = strconv.Atoi(numSensoresStr)
	}

	// Configuração de Atuadores (TCP) - Pedido: ~20
	numAtuadores := 20

	fmt.Println("========================================")
	fmt.Printf(" INICIANDO TESTE DE CARGA HÍBRIDO\n")
	fmt.Printf(" Alvo: %s\n", servidorIP)
	fmt.Printf(" Sensores (UDP): %d | Atuadores 2 (TCP): %d\n", numSensores, numAtuadores)
	fmt.Println("========================================")

	var pacotesEnviados uint64

	// 1. LANÇAR ATUADORES (TCP)
	for i := 1; i <= numAtuadores; i++ {
		go func(id int) {
			addr := fmt.Sprintf("%s:8082", servidorIP)
			conn, err := net.Dial("tcp", addr)
			if err != nil {
				fmt.Printf("[ERRO] Atuador-%d falhou ao conectar: %v\n", id, err)
				return
			}
			defer conn.Close()

			// Registro obrigatório com \n para o bufio.ReadString do servidor
			// Usando identificação do Atuador 2 (Alarme)
			conn.Write([]byte("REGISTRO:ALARME\n"))

			// Mantém a conexão aberta e descarta comandos recebidos (apenas para teste de carga)
			reader := bufio.NewReader(conn)
			for {
				_, err := reader.ReadString('\n')
				if err != nil {
					return
				}
			}
		}(i)
	}

	// 2. LANÇAR SENSORES (UDP)
	for i := 0; i < numSensores; i++ {
		go func(id int) {
			addr := fmt.Sprintf("%s:8080", servidorIP)
			conn, err := net.Dial("udp", addr)
			if err != nil {
				return
			}
			defer conn.Close()

			sensorID := fmt.Sprintf("stress-%04d", id)
			for {
				dados := map[string]interface{}{
					"id":    sensorID,
					"tipo":  "temperatura_reator",
					"valor": 350 + rand.Intn(700),
				}
				msgBytes, _ := json.Marshal(dados)
				conn.Write(msgBytes)
				atomic.AddUint64(&pacotesEnviados, 1)
				time.Sleep(50 * time.Millisecond)
			}
		}(i)
	}

	// Monitoramento
	ticker := time.NewTicker(1 * time.Second)
	for range ticker.C {
		enviados := atomic.SwapUint64(&pacotesEnviados, 0)
		fmt.Printf("[Métrica] UDP: %d pkt/s | Atuadores TCP Ativos: %d\n", enviados, numAtuadores)
	}
}
