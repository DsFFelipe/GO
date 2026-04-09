package main

import (
	"encoding/json"
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
		servidorAddr = "servidor:8082"
	}

	fmt.Printf("Atuador 2 (Sirene) conectando ao Servidor em %s...\n", servidorAddr)

	conn, err := net.Dial("tcp", servidorAddr)
	if err != nil {
		fmt.Printf("Falha crítica ao conectar ao servidor: %v\n", err)
		return
	}
	defer conn.Close()

	// Envia a string de identificação com o delimitador \n
	fmt.Println("Enviando pacote de registro: REGISTRO:ALARME")
	conn.Write([]byte("REGISTRO:ALARME\n"))
	fmt.Println("Conectado e Registrado! Aguardando comandos do broker...")

	// Transmite o estado inicial para sincronizar o painel do cliente
	enviarEstado(conn)

	buffer := make([]byte, 1024)
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			fmt.Println("Conexão com o servidor perdida.")
			break
		}
		comando := strings.ToUpper(strings.TrimSpace(string(buffer[:n])))
		executarComando(comando, conn)
	}
}

func executarComando(comando string, conn net.Conn) {
	mu.Lock()

	if comando == "LIGAR_ALARME" {
		if alarmeLigado {
			fmt.Println("[EMERGÊNCIA] Comando ignorado: LIGAR_ALARME. As sirenes JÁ ESTÃO ATIVADAS.")
		} else {
			alarmeLigado = true
			fmt.Println("[EMERGÊNCIA] ACIONANDO SIRENES! Temperatura do reator crítica.")
		}
	} else if comando == "DESLIGAR_ALARME" {
		if !alarmeLigado {
			fmt.Println("[STATUS] Comando ignorado: DESLIGAR_ALARME. As sirenes já estão desligadas.")
		} else {
			alarmeLigado = false
			fmt.Println("[STATUS] Temperatura estabilizada. Desativando sirenes de emergência.")
		}
	} else {
		fmt.Printf("[AVISO] Comando desconhecido ignorado: %s\n", comando)
		mu.Unlock()
		return
	}

	fmt.Printf("Estado atual da Sirene: Ligada = %v\n", alarmeLigado)
	mu.Unlock()

	// Retorna o feedback do estado atualizado para o servidor
	enviarEstado(conn)
}

func enviarEstado(conn net.Conn) {
	mu.Lock()
	estado := alarmeLigado
	mu.Unlock()

	dados := map[string]interface{}{
		"id":     "ALARME",
		"ligado": estado,
	}

	// Serializa o estado em JSON e adiciona o delimitador TCP de quebra de linha
	b, _ := json.Marshal(dados)
	conn.Write(append(b, '\n'))
}
