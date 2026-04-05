package main

import (
	"bufio"
	"encoding/json" // Importado para decodificar os dados
	"fmt"
	"net"
	"os"
	"strings"
)

// Estrutura que espelha o mapa enviado pelo sensor
type MensagemSensor struct {
	Tipo       string `json:"tipo"`
	Valor      int    `json:"valor"`
	Localidade string `json:"localidade"`
}

func main() {
	comandoChan := make(chan string)

	go testeinput(comandoChan)
	go envia(comandoChan)
	go recebe()

	fmt.Println("Monitor do Cliente em execução...")
	select {}
}

func recebe() {
	endr, err := net.ResolveUDPAddr("udp", ":8080")
	if err != nil {
		return
	}
	conn, err := net.ListenUDP("udp", endr)
	if err != nil {
		return
	}
	defer conn.Close()

	for {
		buffer := make([]byte, 1024)
		n, _, err := conn.ReadFromUDP(buffer)
		if err == nil {
			var dados MensagemSensor
			// Decodifica os bytes (JSON) para a estrutura 'dados'
			err := json.Unmarshal(buffer[:n], &dados)

			if err != nil {
				// Se não for JSON, imprime como texto puro
				fmt.Printf("\n[Msg]: %s\n", string(buffer[:n]))
			} else {
				// Imprime os dados estruturados de forma legível
				fmt.Printf("\n--- Relatório de Telemetria ---")
				fmt.Printf("\nSensor: %s | Local: %s", dados.Tipo, dados.Localidade)
				fmt.Printf("\nValor medido: %d\n", dados.Valor)
			}
		}
	}
}

func testeinput(ch chan<- string) {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("Digite um comando para o atuador: ")
		if !scanner.Scan() {
			break
		}
		msg := scanner.Text()
		ch <- msg
		if strings.ToLower(msg) == "sair" {
			os.Exit(0)
		}
	}
}

func envia(ch <-chan string) {
	for {
		msg := <-ch
		conn, err := net.Dial("tcp", "servidor:8080")
		if err != nil {
			fmt.Printf("Erro ao conectar ao servidor: %v\n", err)
			continue
		}
		conn.Write([]byte(msg))
		conn.Close()
	}
}
