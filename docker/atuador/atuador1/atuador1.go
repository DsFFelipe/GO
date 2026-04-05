package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

func main() {
	// Mantém a escuta para comandos do cliente (TCP) e dados do sensor (UDP)
	go recebesensores()
	go recebecliente()
	fmt.Println("ATUADOR 1 EM OPERAÇÃO - Monitorando Threshold > 50")
	select {}
}

func recebesensores() {
	endr, err := net.ResolveUDPAddr("udp", ":8080")
	if err != nil {
		fmt.Printf("Erro ao resolver UDP: %v\n", err)
		return
	}
	conn, err := net.ListenUDP("udp", endr)
	if err != nil {
		fmt.Printf("Erro ao iniciar escuta UDP: %v\n", err)
		return
	}
	defer conn.Close()

	for {
		buffer := make([]byte, 1024)
		n, _, err := conn.ReadFromUDP(buffer)
		if err == nil {
			msg := strings.TrimSpace(string(buffer[:n]))

			// Converte a mensagem para inteiro para validar a regra de negócio
			valor, err := strconv.Atoi(msg)
			if err != nil {
				// Caso a mensagem não seja um número, apenas exibe como log
				fmt.Printf("[LOG] Mensagem não numérica recebida: %s\n", msg)
				continue
			}

			fmt.Printf("[DADO] Valor recebido do sensor: %d\n", valor)

			// Lógica solicitada: Se maior que 50, executa função de segurança
			if valor > 50 {
				executaAcaoEmergencia(valor)
			}
		}
	}
}

// Simula a função do atuador (Ex: Fechar Barreira de Acesso ou Alerta no PMV)
func executaAcaoEmergencia(v int) {
	fmt.Printf("!!! ALERTA !!! Valor %d excede o limite. EXECUTANDO AÇÃO: [FECHANDO BARREIRA]\n", v)
}

func recebecliente() {
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Printf("Falha no Listen TCP: %v\n", err)
		return
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		buffer := make([]byte, 1024)
		n, _ := conn.Read(buffer)
		fmt.Printf("[COMANDO CLIENTE] Recebido: %s\n", string(buffer[:n]))
		conn.Close()
	}
}
