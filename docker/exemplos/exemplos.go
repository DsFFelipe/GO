package main

import (
	"fmt"
	"net"
	"time"
)

func enviaComponentNomeUDP() {

	conn, err := net.Dial("udp", "QUEMRECEBE:8080")
	if err != nil { // ERRO PADRÃO
		fmt.Printf("Falha ERRO PADRÃO: %v\n", err)
		return
	}
	msg := "MENSAGEM DE QUEM ENVIA"
	i := 0
	for {
		time.Sleep(2 * time.Second)
		_, err = conn.Write([]byte(msg))
		if err != nil { // ERRO PADRÃO
			fmt.Printf("Falha ERRO PADRÃO: %v\n", err)
			//return quebra o código aqui por alguma razão
		}
		fmt.Println("mensagem enviada?")
		i += 1
	}

}

func recebeComponentNomeUDP() {
	fmt.Printf("rrecebeComponentNome")
	endr, err := net.ResolveUDPAddr("udp", ":8080")
	if err != nil {
		fmt.Printf("Falha crítica no recebeComponentUDP: %v\n", err)
		return
	}
	conn, err := net.ListenUDP("udp", endr)
	if err != nil {
		fmt.Printf("Falha crítica no recebeComponentUDP: %v\n", err)
		return
	}

	defer conn.Close()

	for {
		//make trata com slices, mapas e channels, []indica q é um slice
		buffer := make([]byte, 1024)
		n, ComponentEndr, err := conn.ReadFromUDP(buffer)
		if err != nil { // ERRO PADRÃO
			fmt.Printf("Falha ERRO PADRÃO: %v\n", err)
			return
		}
		fmt.Printf("Recebi do Component %s: %s\n", ComponentEndr, string(buffer[:n]))
	}

}
