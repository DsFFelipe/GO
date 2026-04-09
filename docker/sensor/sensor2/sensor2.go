package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"os"
	"time"
)

func main() {
	// Procura o endereço do servidor de telemetria UDP (porto 8080)
	servidorAddr := os.Getenv("SERVIDOR_ADDR")
	if servidorAddr == "" {
		servidorAddr = "servidor:8080"
	}

	conn, err := net.Dial("udp", servidorAddr)
	if err != nil {
		fmt.Printf("Falha crítica ao conectar à matriz de telemetria: %v\n", err)
		return
	}
	defer conn.Close()

	// Inicializa o gerador aleatório para IDs únicos e flutuações de temperatura
	rand.Seed(time.Now().UnixNano())
	sensorID := fmt.Sprintf("reator-temp-%04d", rand.Intn(10000))

	fmt.Printf("Matriz de Temperatura Nuclear Ativa | ID: %s\n", sensorID)

	for {
		// Simula as temperaturas do líquido de refrigeração do reator (280°C a 330°C)
		tempCelsius := 280 + rand.Intn(51)

		// Estrutura os dados removendo completamente a localidade
		dados := map[string]interface{}{
			"id":    sensorID,
			"tipo":  "temperatura_reator",
			"valor": tempCelsius,
		}

		msgBytes, _ := json.Marshal(dados)
		conn.Write(msgBytes)

		// Monitorização local na consola
		if tempCelsius > 320 {
			fmt.Printf("[ALERTA] Temperatura Elevada Detetada: %d°C\n", tempCelsius)
		} else {
			fmt.Printf("[%s] Enviado: %d°C\n", sensorID, tempCelsius)
		}

		// Taxa de envio rápida para monitorização de infraestrutura crítica (1 segundo)
		time.Sleep(3 * time.Second)
	}
}
