package main

import (
	"encoding/json" // Necessário para converter bytes em estruturas de dados
	"fmt"
	"net"
	"sync"
)

// Estrutura para interpretar os dados vindos do sensor
type Dados struct {
	Tipo       string `json:"tipo"`
	Valor      int    `json:"valor"`
	Localidade string `json:"localidade"`
}

var (
	barreiraAberta bool       = true
	mu             sync.Mutex // Garante que apenas uma rotina altere o estado por vez
)

func main() {
	// Canais para comunicação entre goroutines
	sensorParaClienteChan := make(chan []byte)
	DadosSensor := make(chan []byte)
	clienteChan := make(chan []byte)

	// Inicia os fluxos de recebimento e envio
	go recebecliente(clienteChan)
	// Ajustado para usar o canal DadosSensor declarado acima
	go recebesensor(sensorParaClienteChan, DadosSensor)

	go enviacliente(sensorParaClienteChan)

	// Inicia a lógica de decisão automática
	go processarDecisao(DadosSensor, clienteChan)

	go enviaatuadorTCP(clienteChan)

	select {} // Mantém o servidor vivo
}

func recebesensor(chCliente chan<- []byte, DadosSensor chan<- []byte) {
	endr, err := net.ResolveUDPAddr("udp", ":8080")
	if err != nil {
		fmt.Printf("Erro no endereço UDP: %v\n", err)
		return
	}
	conn, err := net.ListenUDP("udp", endr)
	if err != nil {
		fmt.Printf("Erro ao abrir porta UDP: %v\n", err)
		return
	}
	defer conn.Close()

	fmt.Println("Servidor aguardando dados do Sensor (UDP)...")

	for {
		buffer := make([]byte, 1024)
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Printf("Erro na leitura UDP: %v\n", err)
			continue
		}

		// Repassa os bytes brutos para os canais de envio
		chCliente <- buffer[:n]
		DadosSensor <- buffer[:n]
	}
}

func recebecliente(ch chan<- []byte) {
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Printf("Erro no Listen TCP: %v\n", err)
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

func enviaatuadorTCP(ch <-chan []byte) {
	for {
		msg := <-ch
		conn, err := net.Dial("tcp", "atuador1:8080")
		if err != nil {
			continue
		}
		conn.Write(msg)
		conn.Close()
	}
}

func enviacliente(ch <-chan []byte) {
	conn, err := net.Dial("udp", "cliente:8080")
	if err != nil {
		return
	}
	for {
		msg := <-ch
		conn.Write(msg)
	}
}

// Lógica de processamento e decisão automática
func processarDecisao(chSensor <-chan []byte, chAtuador chan<- []byte) {
	for {
		// Recebe os bytes do canal do sensor
		rawBytes := <-chSensor

		var d Dados
		// Converte JSON em estrutura Go
		err := json.Unmarshal(rawBytes, &d)
		if err != nil {
			fmt.Println("Erro ao decodificar JSON:", err)
			continue
		}

		mu.Lock()
		fmt.Printf("\n[TELEMETRIA] %s em %s: %d\n", d.Tipo, d.Localidade, d.Valor)

		// Lógica automática: envia comandos de texto para o atuador via canal
		if d.Valor > 70 && barreiraAberta {
			barreiraAberta = false
			fmt.Println("ALERTA: Nível crítico! Fechando barreira automaticamente.")
			chAtuador <- []byte("FECHAR")
		} else if d.Valor < 50 && !barreiraAberta {
			barreiraAberta = true
			fmt.Println("STATUS: Nível seguro. Abrindo barreira automaticamente.")
			chAtuador <- []byte("ABRIR")
		}

		fmt.Printf("Estado da barreira: Aberta = %v\n", barreiraAberta)
		mu.Unlock()
	}
}
