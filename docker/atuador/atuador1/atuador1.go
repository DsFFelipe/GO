package main

import (
	"encoding/json"
	"fmt"
	"net"
)

// Estrutura para espelhar o JSON do sensor
type DadosSensor struct {
	Tipo       string `json:"tipo"`
	Valor      int    `json:"valor"`
	Localidade string `json:"localidade"`
}

func main() {
	recebeDadosUDP()
}

func recebeDadosUDP() {
	addr, err := net.ResolveUDPAddr("udp", ":8080")
	if err != nil {
		fmt.Printf("Erro ao resolver endereço: %v\n", err)
		return
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Printf("Erro ao abrir socket UDP: %v\n", err)
		return
	}
	defer conn.Close()

	fmt.Println("Atuador aguardando dados via UDP...")

	for {
		buffer := make([]byte, 1024)
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Printf("Erro na leitura: %v\n", err)
			continue
		}

		// Conversão de JSON para a Struct
		var dados DadosSensor
		err = json.Unmarshal(buffer[:n], &dados)
		if err != nil {
			fmt.Printf("Erro ao decodificar JSON: %v\n", err)
			continue
		}

		// Lógica de atuação baseada nos dados recebidos
		processarDecisao(dados)
	}
}

func processarDecisao(d DadosSensor) {
	fmt.Printf("\n[ATUADOR] Dados recebidos de: %s (%s)\n", d.Localidade, d.Tipo)
	fmt.Printf("Índice de chuva: %d\n", d.Valor)

	if d.Valor > 70 {
		fmt.Println("ALERTA: Risco de inundação! Fechando barreiras.")
	} else {
		fmt.Println("Status: Normal.")
	}
}
