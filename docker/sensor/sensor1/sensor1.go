package main

import (
	"bufio"         // Para leitura eficiente de strings no stdin
	"encoding/json" // Para enviar os dados como mapa (JSON)
	"fmt"
	"math/rand"
	"net"
	"os" // Para acessar a entrada padrão (os.Stdin)
	"strings"
	"sync" // Para sincronização de memória (Mutex)
	"time"
)

// Variáveis globais para controle de estado e concorrência
var (
	localidade = "norte"  // Valor inicial padrão
	mu         sync.Mutex // Mutex para evitar condições de corrida
)

func main() {
	fmt.Println("Sensor de Telemetria - Pluviômetro")

	// Executa a função de leitura em uma thread separada (goroutine)
	go inputComandos()

	enviaservidor()
}

// Função para capturar comandos do teclado continuamente
func inputComandos() {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Comando: Digite o novo local e aperte Enter para trocar.")

	for scanner.Scan() {
		texto := strings.TrimSpace(scanner.Text())
		if texto != "" {
			mu.Lock() // Bloqueia o acesso para escrita
			localidade = texto
			mu.Unlock() // Libera o acesso
			fmt.Printf(">>> Localidade alterada para: %s\n", texto)
		}
	}
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
		valorAleatorio := r.Intn(101)

		// Leitura segura da localidade atualizada
		mu.Lock()
		localAtual := localidade
		mu.Unlock()

		// Estrutura em mapa conforme solicitado anteriormente
		mapaDados := map[string]interface{}{
			"tipo":       "pluviometro",
			"valor":      valorAleatorio,
			"localidade": localAtual,
		}

		msgBytes, _ := json.Marshal(mapaDados)

		_, err = conn.Write(msgBytes)
		if err != nil {
			fmt.Printf("Erro ao transmitir dados: %v\n", err)
		} else {
			fmt.Printf("Enviado: [%s] %d unidades\n", localAtual, valorAleatorio)
		}

		time.Sleep(2 * time.Second)
	}
}
