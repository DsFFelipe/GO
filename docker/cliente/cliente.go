package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

func main() {
	// Canal para transferir a string do teclado para a função de envio
	comandoChan := make(chan string)

	go testeinput(comandoChan)
	go envia(comandoChan)
	go recebe()
	fmt.Println("Cliente em execução...")
	select {} // Bloqueia a main para manter as goroutines vivas
}

func testeinput(ch chan<- string) {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Println("\nSelecione um comando para enviar:")
		fmt.Println("1 - Abrir")
		fmt.Println("Ou digite 'sair' para encerrar")

		if !scanner.Scan() {
			break
		}
		msg := scanner.Text()

		// Verifica se o usuário escolheu o comando de fechar barreira
		if msg == "1" {
			ch <- "ABRIR"

		} else {
			// Envia qualquer outra string normalmente
			ch <- msg
		}

		if strings.ToLower(msg) == "sair" {
			os.Exit(0)
		}
	}
}

func envia(ch <-chan string) {
	for {
		// Bloqueia e aguarda uma mensagem chegar pelo canal
		msg := <-ch

		// Estabelece a conexão TCP com o servidor
		conn, err := net.Dial("tcp", "servidor:8080")
		if err != nil {
			fmt.Printf("Erro de conexão: %v\n", err)
			continue
		}

		// Converte a string em um slice de bytes e envia diretamente
		_, err = conn.Write([]byte(msg))
		if err != nil {
			fmt.Printf("Erro ao enviar dados: %v\n", err)
		}

		// Encerra a conexão após o envio
		conn.Close()
	}
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
			fmt.Printf("\n[Recebido]: %s\n", string(buffer[:n]))
		}
	}
}
