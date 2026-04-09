package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	localidade string
	mu         sync.Mutex
	sensorID   string // Identificador único da instância
)

func main() {
	// 1. Busca a localidade inicial nas variáveis de ambiente
	localidade = os.Getenv("SENSOR_LOCAL")
	if localidade == "" {
		localidade = "norte" // Valor padrão de fallback
	}

	// Geração dinâmica do ID baseada no tempo atual para evitar duplicatas
	rand.Seed(time.Now().UnixNano())
	sensorID = fmt.Sprintf("pluv-%04d", rand.Intn(10000))

	fmt.Printf("Sensor 1 Ativo | ID: %s | Local: %s\n", sensorID, localidade)

	go capturarTeclado()
	enviarDados()
}

func capturarTeclado() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		novoLocal := strings.TrimSpace(scanner.Text())
		if novoLocal != "" {
			mu.Lock()
			localidade = novoLocal
			mu.Unlock()
			fmt.Printf(">>> Localidade alterada para: %s\n", novoLocal)
		}
	}
}

func enviarDados() {
	// 2. Busca o endereço do servidor nas variáveis de ambiente
	servidorAddr := os.Getenv("SERVIDOR_ADDR")
	if servidorAddr == "" {
		servidorAddr = "servidor:8080" // Valor padrão de fallback
	}

	conn, err := net.Dial("udp", servidorAddr)
	if err != nil {
		fmt.Printf("Erro na conexão: %v\n", err)
		return
	}
	defer conn.Close()

	for {
		mu.Lock()
		localAtual := localidade
		mu.Unlock()

		valor := rand.Intn(101)

		// O mapa agora inclui o campo "id" para diferenciar instâncias
		dados := map[string]interface{}{
			"id":         sensorID,
			"tipo":       "pluviometro",
			"valor":      valor,
			"localidade": localAtual,
		}

		msgBytes, _ := json.Marshal(dados)
		conn.Write(msgBytes)

		fmt.Printf("[%s] Enviado: %d mm em %s\n", sensorID, valor, localAtual)
		time.Sleep(3 * time.Second)
	}
}
