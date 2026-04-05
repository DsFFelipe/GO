package main

import (
	"fmt"
	"net"
)

func main() {
	// Canais para comunicação entre goroutines
	sensorParaClienteChan := make(chan []byte)
	sensorParaAtuadorChan := make(chan []byte)
	clienteChan := make(chan []byte)

	// Inicia os fluxos de recebimento e envio
	go recebecliente(clienteChan)
	go recebesensor(sensorParaClienteChan, sensorParaAtuadorChan)

	go enviacliente(sensorParaClienteChan)
	go enviaatuadorUDP(sensorParaAtuadorChan)
	go enviaatuadorTCP(clienteChan)

	select {} // Mantém o servidor vivo
}

func recebesensor(chCliente chan<- []byte, chAtuador chan<- []byte) {
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
		chAtuador <- buffer[:n]
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

func enviaatuadorUDP(ch <-chan []byte) {
	conn, err := net.Dial("udp", "atuador1:8080")
	if err != nil {
		return
	}
	for {
		msg := <-ch
		conn.Write(msg)
	}
}
