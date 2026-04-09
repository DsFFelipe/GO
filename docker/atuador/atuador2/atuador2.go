package main

import (
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

var (
	alarmeLigado bool = false
	mu           sync.Mutex
)

func main() {
	servidorAddr := os.Getenv("SERVIDOR_ADDR")
	if servidorAddr == "" {
		servidorAddr = "192.168.0.71:8082"
	}

	fmt.Printf("Atuador de Emergência conectando ao Servidor em %s...\n", servidorAddr)

	conn, err := net.Dial("tcp", servidorAddr)
	if err != nil {
		fmt.Printf("Falha ao conectar ao servidor: %v\n", err)
		return
	}
	defer conn.Close()

	// ---- PROTOCOLO DE HANDSHAKE ----
	// Envia a identificação assim que a conexão TCP é estabelecida
	fmt.Println("Enviando pacote de registro: REGISTRO:ALARME")
	conn.Write([]byte("REGISTRO:ALARME"))
	// --------------------------------

	fmt.Println("Conectado e Registrado! Aguardando comandos do broker...")

	buffer := make([]byte, 1024)

	for {
		n, err := conn.Read(buffer)
		if err != nil {
			fmt.Println("Conexão com o servidor perdida.")
			break
		}

		comando := strings.ToUpper(strings.TrimSpace(string(buffer[:n])))
		executarComando(comando)
	}
}

func executarComando(comando string) {
	mu.Lock()
	defer mu.Unlock()

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
	} else {
		// Como o roteamento agora é exato, se cair aqui é porque o servidor enviou um comando inválido
		fmt.Printf("[AVISO] Comando desconhecido ignorado: %s\n", comando)
	}

	fmt.Printf("Estado atual da Sirene: Ligada = %v\n", alarmeLigado)
}
