package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
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
	Acao string // "ABRIR", "LIGAR_ALARME", etc.
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
	DadosSensor := make(chan []byte)

	chAtuadores := make(chan ComandoAtuador)

	// Passe o canal chAtuadores para o cliente
	go recebecliente(chAtuadores)

	go recebesensor(sensorParaClienteChan, DadosSensor)
	go enviacliente(sensorParaClienteChan)
	go processarDecisao(DadosSensor, chAtuadores)
	go recebeAtuadores()
	go enviaComandosParaAtuadores(chAtuadores)

	select {}
}

// ==========================================
// BROKER DE ATUADORES COM ROTEAMENTO (TCP: 8082)
// ==========================================
func recebeAtuadores() {
	ln, err := net.Listen("tcp", ":8082")
	if err != nil {
		fmt.Printf("Erro no Listen TCP para atuadores: %v\n", err)
		return
	}
	defer ln.Close()

	fmt.Println("Servidor aguardando conexões de Atuadores (TCP na porta 8082)...")

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}

		// Inicia uma goroutine para lidar com o Handshake sem bloquear o Accept
		go func(c net.Conn) {
			buffer := make([]byte, 1024)
			n, err := c.Read(buffer)
			if err != nil {
				c.Close()
				return
			}

			// Lê o primeiro pacote para descobrir quem conectou
			identificacao := strings.TrimSpace(string(buffer[:n]))
			tipo := "DESCONHECIDO"

			if identificacao == "REGISTRO:BARREIRA" {
				tipo = "BARREIRA"
			} else if identificacao == "REGISTRO:ALARME" {
				tipo = "ALARME"
			}

			muAtuadores.Lock()
			atuadoresAtivos[c] = tipo // Registra na tabela de roteamento
			muAtuadores.Unlock()

			fmt.Printf("Novo atuador registrado: %s em %s\n", tipo, c.RemoteAddr().String())
		}(conn)
	}
}

func enviaComandosParaAtuadores(ch <-chan ComandoAtuador) {
	for comando := range ch {
		muAtuadores.Lock()
		for conn, tipo := range atuadoresAtivos {
			// Filtro de Roteamento: Só envia se o alvo bater com o tipo registrado
			if tipo == comando.Alvo {
				_, err := conn.Write([]byte(comando.Acao))
				if err != nil {
					fmt.Printf("Atuador desconectado: %s\n", conn.RemoteAddr().String())
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

		if d.Tipo == "pluviometro" {
			if d.Valor > 95 && barreiraAberta {
				barreiraAberta = false
				fmt.Println("ALERTA: Nível crítico de chuva! Fechando barreira.")
				chAtuador <- ComandoAtuador{Alvo: "BARREIRA", Acao: "FECHAR"}
			} else if d.Valor < 10 && !barreiraAberta {
				barreiraAberta = true
				fmt.Println("STATUS: Nível seguro. Abrindo barreira.")
				chAtuador <- ComandoAtuador{Alvo: "BARREIRA", Acao: "ABRIR"}
			}

		} else if d.Tipo == "temperatura_reator" {
			if d.Valor > 1000 && !alarmeLigado {
				alarmeLigado = true
				fmt.Println("ALERTA CRÍTICO: Temperatura do reator excedeu o limite! Acionando alarme.")
				chAtuador <- ComandoAtuador{Alvo: "ALARME", Acao: "LIGAR_ALARME"}
			} else if d.Valor <= 100 && alarmeLigado {
				alarmeLigado = false
				fmt.Println("STATUS: Temperatura do reator estabilizada. Desligando alarme.")
				chAtuador <- ComandoAtuador{Alvo: "ALARME", Acao: "DESLIGAR_ALARME"}
			}
		}
		mu.Unlock()
	}
}

func recebesensor(chCliente chan<- []byte, DadosSensor chan<- []byte) {
	endr, _ := net.ResolveUDPAddr("udp", "0.0.0.0:8080")
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

func recebecliente(chAtuador chan<- ComandoAtuador) {
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Printf("Erro no Listen TCP para cliente: %v\n", err)
		return
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}

		buffer := make([]byte, 1024)
		n, err := conn.Read(buffer)

		if err == nil {
			// Limpa a string recebida do cliente
			comando := strings.ToUpper(strings.TrimSpace(string(buffer[:n])))
			fmt.Printf("Comando manual do cliente recebido: %s\n", comando)

			// Classifica e roteia o comando de acordo com o protocolo de identificação
			if comando == "ABRIR" || comando == "FECHAR" {
				chAtuador <- ComandoAtuador{Alvo: "BARREIRA", Acao: comando}
			} else if comando == "LIGAR_ALARME" || comando == "DESLIGAR_ALARME" {
				chAtuador <- ComandoAtuador{Alvo: "ALARME", Acao: comando}
			} else {
				fmt.Printf("[AVISO] Comando de cliente não reconhecido pelo roteador: %s\n", comando)
			}
		}

		// O cliente.go foi programado para abrir e fechar a conexão a cada envio
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
