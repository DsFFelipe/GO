package main

import (
	"fmt"
	"math/rand"
	"net"
	"time"
)

func main() {
	fmt.Println("Sensor de Telemetria")
	enviaservidor()
}

func enviaservidor() {
	servidorAddr := "servidor:8080"

	conn, err := net.Dial("udp", servidorAddr)
	if err != nil {
		fmt.Printf("Erro na conexão UDP: %v\n", err)
		return
	}
	defer conn.Close()

	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source)

	for {
		//  Gera um valor  entre 0 e 100
		valorAleatorio := r.Intn(101)

		// Converte o INT para um slice
		// Sprintf para formatar o int como string e depois []byte() para a conversão
		msgBytes := []byte(fmt.Sprintf("%d", valorAleatorio))

		_, err = conn.Write(msgBytes)
		if err != nil {
			fmt.Printf("Erro ao transmitir dados: %v\n", err)
		} else {
			fmt.Printf("Dados enviados: %d unidades\n", valorAleatorio)
		}

		time.Sleep(2 * time.Second)
	}
}
