package main

import (
	"fmt"
	"net"
	"time"
)

func main() {
	fmt.Printf("opasensorqui")
	enviaservidor()
}

func enviaservidor() {

	conn, err := net.Dial("udp", "servidor:8080")
	if err != nil { // ERRO PADRÃO
		fmt.Printf("Falha ERRO PADRÃO: %v\n", err)
		return
	}
	msg := "uiuiui SENSOR 2 PRESENTE"
	i := 0
	for {
		time.Sleep(2 * time.Second)
		_, err = conn.Write([]byte(msg))
		if err != nil { // ERRO PADRÃO
			fmt.Printf("Falha ERRO PADRÃO: %v\n", err)
			//return quebra o código aqui por alguma razão
		}
		fmt.Println("mensagem enviada?")
		i += 1
	}

}
