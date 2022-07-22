package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

func getFile(conn net.Conn, conn2 net.Conn, msg string) {
	log.SetOutput(f)
	log.Println("downloading file", msg)
	conn.Write([]byte(fmt.Sprintf(msg + "\n"))) // отправка сообщения по соединению
	reader := bufio.NewReader(conn2)            // объект чтения из соединения
	stats, err := reader.ReadString('\n')       // получение размера файла
	if err != nil {
		log.Println(err)
		in2 <- "| ошибка соединения"
		return
	}
	stats = strings.TrimSpace(stats)
	statsArr := strings.Fields(stats)
	if len(statsArr) < 3 { // если недостаточно аргументов
		in2 <- "| Файл не существует или ошибка соединения"
		return
	}
	fileSize, err := strconv.ParseInt(statsArr[2], 10, 64) // получение размера файла для создания
	if err != nil || fileSize == -1 {
		log.Println("file size error", err)
		in2 <- "| Ошибка открытия файла или файл не существует "
		return
	}
	conn.Write([]byte(fmt.Sprintf("/startdsend %s\n", strings.TrimPrefix(msg, "/download ")))) // подтверждение закачки
	buf := new(bytes.Buffer)                                                                   // баффер для чтения из соединения
	io.Copy(buf, io.LimitReader(conn2, fileSize))                                              // запись в буфер из соединения
	outputFile, err := os.Create(rootDownload + "/" + strings.TrimPrefix(msg, "/download "))   // создание файла
	if err != nil {
		log.Println(err)
	}
	io.Copy(outputFile, bytes.NewReader(buf.Bytes())) // запись из буфера в файл
	defer outputFile.Close()
	in2 <- fmt.Sprintf("| Файл %v скачан", strings.TrimPrefix(msg, "/download")) // обновление статуса
	log.Println("file downloaded successfully")
}

func sendFile(conn net.Conn, conn2 net.Conn, msg string) {
	log.SetOutput(f)
	log.Println("uploading file", msg)
	args := strings.Fields(msg)
	if len(args) < 2 {
		in2 <- "| Неправильный формат команды."
		return
	}
	data, err := ioutil.ReadFile(rootUpload + "/" + args[1]) // чтение файла для отправки
	if err != nil {
		in2 <- "| Ошибка открытия файла " + err.Error()
		log.Println(err)
		return
	}
	conn.Write([]byte(fmt.Sprintf("%s %d\n", msg, len(data)))) // отпрвка размера и команды на сервер
	reader := bufio.NewReader(conn2)
	incoming, err := reader.ReadString('\n')
	if err != nil {
		log.Println(err)
		in2 <- "| ошибка соединения"
		return
	}
	incoming = strings.TrimSpace(incoming)
	statsArr := strings.Fields(incoming)
	if statsArr[0] != "200" { // если не пришел код подтверждения
		in2 <- "| Ошибка при загрузке " + incoming
		log.Println(incoming)
		return
	}
	io.Copy(conn2, bytes.NewReader(data)) // запись файла в соединение
	incoming, err = reader.ReadString('\n')
	in2 <- incoming // вывод сообщения об окончании
	log.Println("finished uploading file", msg)
}

func listFiles(conn net.Conn, conn2 net.Conn, msg string) {
	log.Println("starting listing files", msg)
	f, err := os.OpenFile("testlogfile", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()
	conn.Write([]byte(fmt.Sprintf(msg + "\n"))) // отрправка команды
	reader := bufio.NewReader(conn2)
	list, err := reader.ReadString('|') // чтение до символа окончания
	list = strings.TrimSpace(list)
	in2 <- list[:len(list)-1] // вывод
	log.Println("finished listing file", msg)
}
