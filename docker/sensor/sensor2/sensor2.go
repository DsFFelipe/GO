package main

import (
	"fmt"
	"math/rand"
	"net"
	"time"
)

const (
	SERVER_ADDR = "servidor:8081" // Endereço do Servidor Central
	SENSOR_TYPE = "UV_SENSOR"     // Identificador do tipo
)

func main() {
	fmt.Printf("[%s] Iniciando telemetria...\n", SENSOR_TYPE)

	// Resolve o endereço do servidor
	addr, err := net.ResolveUDPAddr("udp", SERVER_ADDR)
	if err != nil {
		fmt.Printf("Erro ao resolver endereço: %v\n", err)
		return
	}

	// Cria a conexão UDP
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		fmt.Printf("Erro ao conectar via UDP: %v\n", err)
		return
	}
	defer conn.Close()

	// Loop de envio de dados
	for {
		// Gera valor inteiro de Radiação UV (Escala 0 a 15 para cobrir extremos)
		valorUV := rand.Intn(16)

		// Formata a mensagem: "TIPO:VALOR" para facilitar o parsing no servidor
		msg := fmt.Sprintf("%s:%d", SENSOR_TYPE, valorUV)

		_, err := conn.Write([]byte(msg))
		if err != nil {
			fmt.Printf("Erro ao enviar dado: %v\n", err)
		} else {
			fmt.Printf("[SENT] %s -> Valor: %d\n", SENSOR_TYPE, valorUV)
		}

		// Intervalo de leitura (ex: 3 segundos para não sobrecarregar o log)
		time.Sleep(3 * time.Second)
	}
}
