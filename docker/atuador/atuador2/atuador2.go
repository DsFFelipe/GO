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
	// alarmeLigado armazena o estado binário do atuador.
	alarmeLigado bool = false
	// mu garante a atomicidade das operações sobre o estado global,
	// prevenindo condições de corrida entre a leitura e a escrita.
	mu sync.Mutex
)

func main() {
	// Configuração de endpoint via variável de ambiente (padrão Cloud-Native).
	servidorAddr := os.Getenv("SERVIDOR_ADDR")
	if servidorAddr == "" {
		servidorAddr = "servidor:8082"
	}

	fmt.Printf("Atuador 2 (Sirene) conectando ao Servidor em %s...\n", servidorAddr)

	// Estabelecimento da conexão de fluxo (Stream) TCP.
	conn, err := net.Dial("tcp", servidorAddr)
	if err != nil {
		fmt.Printf("Falha crítica ao conectar ao servidor: %v\n", err)
		return
	}
	defer conn.Close()

	// Handshake/Registro: Identifica este nó para o servidor/broker.
	// O uso do sufixo \n é vital para que o receptor identifique o fim da mensagem.
	fmt.Println("Enviando pacote de registro: REGISTRO:ALARME")
	conn.Write([]byte("REGISTRO:ALARME\n"))
	fmt.Println("Conectado e Registrado! Aguardando comandos do broker...")

	// Sincronização inicial: informa ao servidor o estado atual assim que a conexão sobe.
	enviarEstado(conn)

	// Loop principal de escuta.
	buffer := make([]byte, 1024)
	for {
		// Read é uma operação bloqueante que aguarda dados no socket.
		n, err := conn.Read(buffer)
		if err != nil {
			fmt.Println("Conexão com o servidor perdida. Encerrando atuador.")
			break
		}

		// Normalização da mensagem: remove ruídos de buffer e padroniza o comando.
		comando := strings.ToUpper(strings.TrimSpace(string(buffer[:n])))
		executarComando(comando, conn)
	}
}

// executarComando implementa a lógica de controle e transição de estado.
func executarComando(comando string, conn net.Conn) {
	mu.Lock()

	// Implementação de verificações de estado para evitar redundância (Idempotência).
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

	// Feedback em malha fechada: o servidor só tem certeza da atuação após este envio.
	enviarEstado(conn)
}

// enviarEstado serializa o estado atual em JSON para telemetria.
func enviarEstado(conn net.Conn) {
	mu.Lock()
	estado := alarmeLigado
	mu.Unlock()

	// Estruturação dos dados para conformidade com o parser do servidor.
	dados := map[string]interface{}{
		"id":     "ALARME",
		"ligado": estado,
	}

	// Marshaling: conversão da estrutura de memória para representação textual JSON.
	b, _ := json.Marshal(dados)
	// Adição de Delimitador de Linha: Essencial para protocolos baseados em stream TCP
	// para evitar o problema de "mensagens coladas" (Message Fragmentation).
	conn.Write(append(b, '\n'))
}
