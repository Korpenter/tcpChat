package main

import (
	"fmt"
	"os"
	"tcpSocketChat/tcpChatServer"
	"time"
)

func main() {
	var port int
	if _, err := os.Stat(tcpChatServer.FileStorageRoot); os.IsNotExist(err) {
		if err = os.Mkdir(tcpChatServer.FileStorageRoot, 0777); err != nil {
			fmt.Println(err)
			time.Sleep(time.Second * 3)
			return
		}
	}
	fmt.Println("Введите порт для запуска сервера: ") // выбор порта для сервера
	fmt.Scanln(&port)                                 // ввод порта
	tcpChatServer.Start(port)                         // запуск сервера
	for {
	}
}
