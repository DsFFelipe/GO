package main

import (
	"fmt"
	"net"
)

func main() {
	go recebesensores()
	go recebecliente()
	fmt.Println("ATUADOR EXECUTOU.")
	select {}
}

func recebecliente() {
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

		// _ quer dizer que nega erros
		// := é uma declaração curta imutável que pode ser feita apenas dentro de funções
		// o for é como se fosse um while true
		n, _ := conn.Read(buffer)
		// Exibe a mensagem convertendo os bytes lidos em string
		fmt.Printf("ATUADOR RECEBEU, cliente: %s\n", string(buffer[:n]))
		// Fecha a conexão com o cliente atual
		conn.Close()
	}
}

func recebesensores() {
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
			fmt.Printf("\n[SOU ATUADOR E FOI Recebido]: %s\n", string(buffer[:n]))
		}
	}
}
