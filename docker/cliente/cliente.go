package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

// Estrutura que espelha o mapa enviado pelo sensor
type MensagemSensor struct {
	Tipo       string `json:"tipo"`
	Valor      int    `json:"valor"`
	Localidade string `json:"localidade"`
}

// Variáveis globais para armazenar o estado dos sensores
var (
	sensoresAtivos = make(map[string]MensagemSensor)
	mu             sync.Mutex
)

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
			err := json.Unmarshal(buffer[:n], &dados)

			if err != nil {
				// Ignora mensagens que não sejam JSON para manter o terminal limpo
				continue
			} else {
				// Lock para escrita segura na variável global
				mu.Lock()
				chave := fmt.Sprintf("%s-%s", dados.Tipo, dados.Localidade)
				sensoresAtivos[chave] = dados
				mu.Unlock()
			}
		}
	}
}

func testeinput(ch chan<- string) {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("Digite um comando (ou 'listar' para ver sensores): ")
		if !scanner.Scan() {
			break
		}
		msg := strings.TrimSpace(scanner.Text())

		// Intercepta o comando de listagem localmente
		if strings.ToLower(msg) == "listar" {
			exibirSensores()
			continue
		}

		ch <- msg
		if strings.ToLower(msg) == "sair" {
			os.Exit(0)
		}
	}
}

func exibirSensores() {
	// Lock para leitura segura da variável global
	mu.Lock()
	defer mu.Unlock()

	fmt.Println("\n--- Relatório de Telemetria ---")
	if len(sensoresAtivos) == 0 {
		fmt.Println("Nenhum dado recebido ainda.")
	} else {
		for _, dados := range sensoresAtivos {
			fmt.Printf("Sensor: %s | Local: %s | Valor medido: %d\n", dados.Tipo, dados.Localidade, dados.Valor)
		}
	}
	fmt.Println("-------------------------------")
}

func envia(ch <-chan string) {
	for {
		msg := <-ch
		conn, err := net.Dial("tcp", "servidor:8080")
		if err != nil {
			fmt.Printf("\nErro ao conectar ao servidor: %v\n", err)
			continue
		}
		conn.Write([]byte(msg))
		conn.Close()
	}
}
