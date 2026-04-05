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

// Variáveis globais para controle de estado
var (
	localidade = "norte"  // Valor inicial
	mu         sync.Mutex // Proteção para acesso simultâneo
)

func main() {
	fmt.Println("Sensor 1: Pluviômetro ativo")

	// Iniciamos a captura de teclado em segundo plano
	go capturarTeclado()

	// Iniciamos o envio contínuo de dados
	enviarDados()
}

// capturarTeclado lê a entrada do terminal (os.Stdin)
func capturarTeclado() {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Digite a nova localidade e pressione Enter:")

	for scanner.Scan() {
		novoLocal := strings.TrimSpace(scanner.Text())
		if novoLocal != "" {
			mu.Lock() // Bloqueia a variável para alteração segura
			localidade = novoLocal
			mu.Unlock() // Libera a variável
			fmt.Printf(">>> Localidade alterada para: %s\n", novoLocal)
		}
	}
}

// enviarDados realiza a transmissão UDP para o servidor
func enviarDados() {
	servidorAddr := "servidor:8080"
	conn, err := net.Dial("udp", servidorAddr)
	if err != nil {
		fmt.Printf("Erro na conexão: %v\n", err)
		return
	}
	defer conn.Close()

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for {
		// Leitura segura da variável compartilhada
		mu.Lock()
		localAtual := localidade
		mu.Unlock()

		valor := r.Intn(101) // Simula índice de chuva

		// Criamos o mapa (JSON) para envio
		dados := map[string]interface{}{
			"tipo":       "pluviometro",
			"valor":      valor,
			"localidade": localAtual,
		}

		msgBytes, _ := json.Marshal(dados)
		conn.Write(msgBytes)

		fmt.Printf("Enviado: %s (%d) de %s\n", "pluviometro", valor, localAtual)
		time.Sleep(2 * time.Second)
	}
}
