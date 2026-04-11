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
	// localidade é uma variável compartilhada entre goroutines.
	localidade string
	// mu protege o acesso à variável 'localidade' contra condições de corrida.
	mu sync.Mutex
	// sensorID garante que as mensagens sejam rastreáveis no servidor.
	sensorID string
)

func main() {
	// Inicialização de Contexto: define onde o sensor está operando.
	localidade = os.Getenv("SENSOR_LOCAL")
	if localidade == "" {
		localidade = "norte"
	}

	// Semente para o gerador de números pseudo-aleatórios.
	// Em Go 1.20+, o rand.Seed é gerenciado automaticamente, mas aqui garante unicidade do ID.
	rand.Seed(time.Now().UnixNano())
	sensorID = fmt.Sprintf("geiger-%04d", rand.Intn(10000))

	fmt.Printf("Sensor 1 Ativo | ID: %s | Local: %s\n", sensorID, localidade)

	// Concorrência: Inicia uma thread leve para escutar o teclado.
	// Isso permite que o programa faça duas coisas ao mesmo tempo: ler input e enviar dados.
	go capturarTeclado()

	// Inicia o loop principal de transmissão.
	enviarDados()
}

func capturarTeclado() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		novoLocal := strings.TrimSpace(scanner.Text())
		if novoLocal != "" {
			// Seção Crítica: Bloqueia o Mutex antes de alterar o valor compartilhado.
			mu.Lock()
			localidade = novoLocal
			mu.Unlock()
			fmt.Printf(">>> Localidade alterada para: %s\n", novoLocal)
		}
	}
}

func enviarDados() {
	servidorAddr := os.Getenv("SERVIDOR_ADDR")
	if servidorAddr == "" {
		servidorAddr = "servidor:8080"
	}

	// Protocolo UDP
	//  UDP não estabelece uma conexão persistente (handshake).
	conn, err := net.Dial("udp", servidorAddr)
	if err != nil {
		fmt.Printf("Erro na conexão: %v\n", err)
		return
	}
	defer conn.Close()

	for {
		// Proteção de leitura: Garante que não leremos a localidade enquanto ela é alterada no teclado.
		mu.Lock()
		localAtual := localidade
		mu.Unlock()

		// Simulação de leitura de sensor (0 a 100 microsieverts/h).
		valor := rand.Intn(101)

		dados := map[string]interface{}{
			"id":         sensorID,
			"tipo":       "Geiger",
			"valor":      valor,
			"localidade": localAtual,
		}

		// Serialização do payload para JSON.
		msgBytes, _ := json.Marshal(dados)

		// Envio sem confirmação (Fire-and-Forget).
		conn.Write(msgBytes)

		fmt.Printf("[%s] Enviado: %d microsievert por hora em %s\n", sensorID, valor, localAtual)

		// Intervalo de amostragem de 1 segundos (Duty Cycle).
		time.Sleep(1 * time.Second)
	}
}
