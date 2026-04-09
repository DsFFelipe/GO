package main

import (
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

// Estado global do alarme e controle de concorrência
var (
	alarmeLigado bool       = false
	mu           sync.Mutex // Previne condições de corrida
)

func main() {
	servidorAddr := os.Getenv("SERVIDOR_ADDR")
	if servidorAddr == "" {
		servidorAddr = "192.168.0.71:8082"
	}

	fmt.Printf("Atuador de Emergência conectando ao Broker TCP em %s...\n", servidorAddr)

	conn, err := net.Dial("tcp", servidorAddr)
	if err != nil {
		fmt.Printf("Falha crítica ao conectar ao servidor: %v\n", err)
		return
	}
	defer conn.Close()

	fmt.Println("Conectado! Aguardando sinais do reator nuclear...")

	buffer := make([]byte, 1024)

	// Loop bloqueante para escuta contínua de pacotes TCP
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			fmt.Println("Conexão TCP com o servidor encerrada inesperadamente.")
			break
		}

		comando := strings.ToUpper(strings.TrimSpace(string(buffer[:n])))
		executarComando(comando)
	}
}

func executarComando(comando string) {
	mu.Lock()
	defer mu.Unlock()

	// Filtro na camada de aplicação: processa apenas os sinais do reator
	if comando == "LIGAR_ALARME" {
		if alarmeLigado {
			fmt.Println("[EMERGÊNCIA] Comando recebido: LIGAR_ALARME. As sirenes JÁ ESTÃO ATIVADAS.")
		} else {
			alarmeLigado = true
			fmt.Println("[EMERGÊNCIA] ACIONANDO SIRENES! Temperatura do reator crítica.")
		}
	} else if comando == "DESLIGAR_ALARME" {
		if !alarmeLigado {
			fmt.Println("[STATUS] Comando recebido: DESLIGAR_ALARME. As sirenes já estão desligadas.")
		} else {
			alarmeLigado = false
			fmt.Println("[STATUS] Temperatura estabilizada. Desativando sirenes de emergência.")
		}
	} else if comando == "ABRIR" || comando == "FECHAR" {
		// Ignora silenciosamente os comandos do pluviômetro para não poluir o log
	} else {
		fmt.Printf("[AVISO] Payload de rede desconhecido ignorado: %s\n", comando)
	}

	// Exibe o estado da máquina de estados deste atuador
	if comando == "LIGAR_ALARME" || comando == "DESLIGAR_ALARME" {
		fmt.Printf("Estado atual da Sirene: Ligada = %v\n", alarmeLigado)
	}
}
