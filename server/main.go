package main

import (
	"fmt"
	"tcpSocketChat/tcpChatServer"
)

func main() {
	var port int
	fmt.Println("Введите порт для запуска сервера: ") // выбор порта для сервера
	fmt.Scanln(&port)                                 // ввод порта
	tcpChatServer.Start(port)                         // запуск сервера
	for {
	}
}
