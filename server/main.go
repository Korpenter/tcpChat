package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"tcpSocketChat/tcpChatServer"
	"time"
)

var port = flag.Int("port", 8888, "Порт") // флаг порта, если не введен, 8888

func main() {
	var err error
	if _, err = os.Stat(tcpChatServer.FileStorageRoot); os.IsNotExist(err) {
		if err = os.Mkdir(tcpChatServer.FileStorageRoot, 0777); err != nil {
			fmt.Println(err)
			time.Sleep(time.Second * 3)
			return
		}
	}

	fmt.Println("Запуск сервера на порте: ", *port)
	tcpChatServer.Start(*port)                                                                    // запуск сервера
	tcpChatServer.F, err = os.OpenFile("serverLogs.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666) // открытие для логов
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer tcpChatServer.F.Close()
	for {
	}
}
