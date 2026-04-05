package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"time" // Adicionado para controle de timeout
)

func main() {
	// Canal para transferir a string do teclado para a função de envio
	// Alteração: Adicionado buffer de 100 para evitar que o input bloqueie se a rede estiver lenta
	comandoChan := make(chan string, 100)

	go testeinput(comandoChan)
	go envia(comandoChan)
	go recebe()
	fmt.Println("Cliente em execução...")
	select {} // Bloqueia a main para manter as goroutines vivas
}

func testeinput(ch chan<- string) {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("\nDigite o comando: ")
		// Scan() bloqueia a execução esperando o Enter
		if !scanner.Scan() {
			break
		}
		msg := scanner.Text()

		// Envia para a goroutine de rede via canal
		ch <- msg

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
		// Alteração: DialTimeout adicionado para não travar o cliente caso o servidor esteja sobrecarregado
		conn, err := net.DialTimeout("tcp", "servidor:8080", 3*time.Second)
		if err != nil {
			fmt.Printf("\n[ERRO] Servidor ocupado ou inacessível: %v\n", err)
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
			// Alteração: Uso de \r e re-impressão do prompt para limpar a linha e manter o contexto do usuário
			fmt.Printf("\r[Recebido]: %s\nDigite o comando: ", string(buffer[:n]))
		}
	}
}
