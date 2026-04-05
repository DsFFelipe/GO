package main

import (
	"fmt"
	"net"
)

func main() {
	// Canais independentes
	sensorParaClienteChan := make(chan []byte)
	sensorParaAtuadorChan := make(chan []byte)
	clienteChan := make(chan []byte)

	go recebecliente(clienteChan)
	go recebesensor1(sensorParaClienteChan, sensorParaAtuadorChan)

	go enviacliente(sensorParaClienteChan)
	go enviaatuador1UDP(sensorParaAtuadorChan)
	go enviaatuador1TCP(clienteChan)

	select {}

}

func recebesensor1(chCliente chan<- []byte, chAtuador chan<- []byte) {
	fmt.Printf("recebesensor1aqui")
	endr, err := net.ResolveUDPAddr("udp", ":8080")
	if err != nil {

		fmt.Printf("Falha crítica no recebesensor: %v\n", err)
		return
	}
	conn, err := net.ListenUDP("udp", endr)
	if err != nil {
		fmt.Printf("Falha crítica no recebesensor: %v\n", err)
		return
	}

	defer conn.Close()

	for {
		buffer := make([]byte, 1024)
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil { // ERRO PADRÃO
			fmt.Printf("Falha ERRO PADRÃO: %v\n", err)
			return
		}

		chCliente <- buffer[:n]
		chAtuador <- buffer[:n]
	}

}

func recebecliente(ch chan<- []byte) {
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		// I
		fmt.Printf("Falha crítica no Listen: %v\n", err)
		return
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Erro ao aceitar conexão:", err)
			continue //
		}
		buffer := make([]byte, 1024)

		n, _ := conn.Read(buffer)
		ch <- buffer[:n]

		fmt.Printf("Mensagem recebida, cliente: %s\n", string(buffer[:n]))

		conn.Close()
	}
}

func enviaatuador1TCP(ch <-chan []byte) {
	for {
		msg := <-ch

		conn, err := net.Dial("tcp", "atuador1:8080")
		if err != nil {
			fmt.Printf("Erro de conexão: %v\n", err)
			continue
		}

		_, err = conn.Write([]byte(msg))
		if err != nil {
			fmt.Printf("Erro ao enviar dados: %v\n", err)
		}

		conn.Close()
	}
}

func enviacliente(ch <-chan []byte) {
	conn, err := net.Dial("udp", "cliente:8080")
	if err != nil {
		fmt.Printf("Falha ERRO PADRÃO: %v\n", err)
		return
	}

	for {
		msg := <-ch
		_, err = conn.Write(msg)

		if err != nil {
			fmt.Printf("Falha ERRO PADRÃO: %v\n", err)
		}
	}
}

func enviaatuador1UDP(ch <-chan []byte) {
	conn, err := net.Dial("udp", "atuador1:8080")
	if err != nil {
		fmt.Printf("Falha ERRO PADRÃO: %v\n", err)
		return
	}

	for {
		msg := <-ch
		_, err = conn.Write(msg)

		if err != nil {
			fmt.Printf("Falha ERRO PADRÃO: %v\n", err)
		}
	}
}
