package main

import (
	"encoding/json"
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

	conn, err := net.Dial("tcp", servidorAddr)
	if err != nil {
		return
	}
	defer conn.Close()

	conn.Write([]byte("REGISTRO:ALARME\n"))
	enviarEstado(conn)

	buffer := make([]byte, 1024)
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			break
		}
		comando := strings.ToUpper(strings.TrimSpace(string(buffer[:n])))
		executarComando(comando, conn)
	}
}

func executarComando(comando string, conn net.Conn) {
	mu.Lock()
	if comando == "LIGAR_ALARME" {
		alarmeLigado = true
	} else if comando == "DESLIGAR_ALARME" {
		alarmeLigado = false
	} else {
		mu.Unlock()
		return
	}
	mu.Unlock()
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
	b, _ := json.Marshal(dados)
	conn.Write(append(b, '\n'))
}
