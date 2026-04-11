package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

type Dados struct {
	ID         string `json:"id"`
	Tipo       string `json:"tipo"`
	Valor      int    `json:"valor"`
	Localidade string `json:"localidade"`
}

// Estrutura para o roteamento interno de comandos
type ComandoAtuador struct {
	Alvo string // "BARREIRA" ou "ALARME"
	Acao string // "ABRIR", "LIGAR_ALARME"
}

var (
	barreiraAberta bool = true
	alarmeLigado   bool = false
	mu             sync.Mutex

	// Tabela de roteamento: mapeia a conexão de rede ao TIPO do atuador
	atuadoresAtivos = make(map[net.Conn]string)
	muAtuadores     sync.Mutex
)

func main() {
	sensorParaClienteChan := make(chan []byte)

	DadosSensor := make(chan []byte, 100)

	chAtuadores := make(chan ComandoAtuador)

	// Canal para o pipeline de estados dos atuadores
	estadoAtuadorChan := make(chan []byte)

	go recebecliente(chAtuadores)
	go recebesensor(sensorParaClienteChan, DadosSensor)
	go enviacliente(sensorParaClienteChan) // Mantém envio UDP de sensores
	go processarDecisao(DadosSensor, chAtuadores)

	//Thread Pool
	for i := 0; i < 3; i++ {
		go processarDecisao(DadosSensor, chAtuadores)
	}

	go recebeAtuadores(estadoAtuadorChan)
	go enviaComandosParaAtuadores(chAtuadores)
	go enviaEstadoTCP(estadoAtuadorChan) // Novo envio TCP de estados

	select {}
}

func recebeAtuadores(chEstado chan<- []byte) {
	port := os.Getenv("ATUADOR_PORT")
	if port == "" {
		port = "8082"
	}

	ln, err := net.Listen("tcp", ":"+port)

	if err != nil {
		fmt.Printf("Erro no Listen TCP Atuadores: %v\n", err)
		return
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}

		go func(c net.Conn) {
			reader := bufio.NewReader(c)

			// Lê o handshake até o delimitador \n
			linha, err := reader.ReadString('\n')
			if err != nil {
				c.Close()
				return
			}

			identificacao := strings.TrimSpace(linha)
			tipo := "DESCONHECIDO"
			if identificacao == "REGISTRO:BARREIRA" {
				tipo = "BARREIRA"
			} else if identificacao == "REGISTRO:ALARME" {
				tipo = "ALARME"
			}

			muAtuadores.Lock()
			atuadoresAtivos[c] = tipo
			muAtuadores.Unlock()

			// Loop contínuo para ouvir os estados deste atuador específico
			for {
				pacoteEstado, err := reader.ReadBytes('\n')
				if err != nil {
					c.Close()
					muAtuadores.Lock()
					delete(atuadoresAtivos, c)
					muAtuadores.Unlock()
					break
				}
				chEstado <- pacoteEstado
			}
		}(conn)
	}
}

// Estabelece conexão TCP pontual para garantir a entrega do estado crítico
func enviaEstadoTCP(ch <-chan []byte) {
	clienteTCP := os.Getenv("CLIENTE_TCP_ADDR")
	if clienteTCP == "" {
		clienteTCP = "192.168.0.118:8084"
	}

	for msg := range ch {
		// Timeout de Dial para não bloquear o servidor caso o cliente monitor caia
		conn, err := net.DialTimeout("tcp", clienteTCP, 2*time.Second)
		if err == nil {
			conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
			conn.Write(msg)
			conn.Close() // Fecha após envio (comportamento de webhook/push)
		}
	}
}

// ==========================================
// BROKER DE ATUADORES COM ROTEAMENTO (TCP: 8082)
// ==========================================

func enviaComandosParaAtuadores(ch <-chan ComandoAtuador) {
	for comando := range ch {
		muAtuadores.Lock()
		for conn, tipo := range atuadoresAtivos {
			// Filtro de Roteamento: Só envia se o alvo bater com o tipo registrado
			if tipo == comando.Alvo {
				conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
				_, err := conn.Write([]byte(comando.Acao))
				if err != nil {
					fmt.Printf("Atuador desconectado (falha de escrita): %s\n", conn.RemoteAddr().String())
					conn.Close()
					delete(atuadoresAtivos, conn)
				}
			}
		}
		muAtuadores.Unlock()
	}
}

// ==========================================
// LÓGICA DE SENSORES
// ==========================================
func processarDecisao(chSensor <-chan []byte, chAtuador chan<- ComandoAtuador) {
	for {
		rawBytes := <-chSensor

		var d Dados
		err := json.Unmarshal(rawBytes, &d)
		if err != nil {
			continue
		}

		mu.Lock()

		if d.Tipo == "Geiger" {
			if d.Valor > 95 && barreiraAberta {
				barreiraAberta = false
				fmt.Println("ALERTA: Nível crítico de radiação! Fechando barreira.")
				chAtuador <- ComandoAtuador{Alvo: "BARREIRA", Acao: "FECHAR"}
			} else if d.Valor < 10 && !barreiraAberta {
				barreiraAberta = true
				fmt.Println("STATUS: Nível seguro. Abrindo barreira.")
				chAtuador <- ComandoAtuador{Alvo: "BARREIRA", Acao: "ABRIR"}
			}

		} else if d.Tipo == "temperatura_reator" {
			if d.Valor > 600 && !alarmeLigado {
				alarmeLigado = true
				fmt.Println("ALERTA CRÍTICO: Temperatura do reator excedeu o limite! Acionando alarme.")
				chAtuador <- ComandoAtuador{Alvo: "ALARME", Acao: "LIGAR_ALARME"}
			} else if d.Valor <= 500 && alarmeLigado {
				alarmeLigado = false
				fmt.Println("STATUS: Temperatura do reator estabilizada. Desligando alarme.")
				chAtuador <- ComandoAtuador{Alvo: "ALARME", Acao: "DESLIGAR_ALARME"}
			}
		}
		mu.Unlock()
	}
}

func recebesensor(chCliente chan<- []byte, DadosSensor chan<- []byte) {
	port := os.Getenv("SENSOR_PORT")
	if port == "" {
		port = "8080"
	} // Fallback

	endr, _ := net.ResolveUDPAddr("udp", "0.0.0.0:"+port)
	conn, _ := net.ListenUDP("udp", endr)
	defer conn.Close()
	fmt.Println("Servidor aguardando dados do Sensor (UDP na porta 8080)...")
	for {
		buffer := make([]byte, 1024)
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			continue
		}
		chCliente <- buffer[:n]
		DadosSensor <- buffer[:n]
	}
}

// ==========================================
// RECEÇÃO DE COMANDOS DO CLIENTE (TCP) COM PROTEÇÃO DE FALHAS
// ==========================================
func recebecliente(chAtuador chan<- ComandoAtuador) {
	port := os.Getenv("CLIENTE_PORT")
	if port == "" {
		port = "8080"
	}

	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Printf("Erro no Listen TCP para cliente: %v\n", err)
		return
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}

		// IMPLEMENTAÇÃO DE DEADLINE: Protege o servidor contra clientes "zombies"
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))

		buffer := make([]byte, 1024)
		n, err := conn.Read(buffer)

		if err == nil {
			comando := strings.ToUpper(strings.TrimSpace(string(buffer[:n])))
			fmt.Printf("Comando manual do cliente recebido: %s\n", comando)

			// Determinar o alvo lógico
			alvo := "DESCONHECIDO"
			if comando == "ABRIR" || comando == "FECHAR" {
				alvo = "BARREIRA"
			} else if comando == "LIGAR_ALARME" || comando == "DESLIGAR_ALARME" {
				alvo = "ALARME"
			}

			// VERIFICAÇÃO DE ESTADO: Avalia se o atuador alvo está realmente registado e online
			muAtuadores.Lock()
			atuadorOnline := false
			for _, tipo := range atuadoresAtivos {
				if tipo == alvo {
					atuadorOnline = true
					break
				}
			}
			muAtuadores.Unlock()

			// Feedback direto e síncrono para a aplicação cliente
			conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
			if !atuadorOnline && alvo != "DESCONHECIDO" {
				msgErro := fmt.Sprintf("FALHA: Nao e possivel executar '%s'. O atuador %s esta DESCONECTADO.\n", comando, alvo)
				conn.Write([]byte(msgErro))
			} else if alvo != "DESCONHECIDO" {
				// Atualiza o estado interno e encarrega o broker de rotear
				mu.Lock()
				if alvo == "BARREIRA" {
					barreiraAberta = (comando == "ABRIR")
				} else if alvo == "ALARME" {
					alarmeLigado = (comando == "LIGAR_ALARME")
				}
				mu.Unlock()

				chAtuador <- ComandoAtuador{Alvo: alvo, Acao: comando}
				conn.Write([]byte("SUCESSO: Comando aceite e roteado para " + alvo + "\n"))
			} else {
				conn.Write([]byte("ERRO: Comando nao reconhecido.\n"))
			}
		}

		conn.Close()
	}
}

func enviacliente(ch <-chan []byte) {
	clienteAddr := os.Getenv("CLIENTE_ADDR")
	if clienteAddr == "" {
		clienteAddr = "192.168.0.118:8081"
	}
	conn, _ := net.Dial("udp", clienteAddr)
	for {
		msg := <-ch
		if conn != nil {
			conn.Write(msg)
		}
	}
}
