package main

import (
	"fmt"
	"tcpSocketChat/tcpServer"
)

func main() {
	var port int
	fmt.Println("Введите порт для запуска сервера: ") // выбор порта для сервера
	fmt.Scanln(&port)                                 // ввод порта
	tcpServer.Start(port)                             // запуск сервера
	for {
	}
}
