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

	f, err := os.OpenFile("testlogfile", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()
	conn.Write([]byte(fmt.Sprintf(msg + "\n")))

	log.Println("\treader := bufio.NewReader(conn)")
	reader := bufio.NewReader(conn2)
	log.Println("\tmsg, err := reader.ReadString('\\n')")
	stats, err := reader.ReadString('\n')
	log.Println("\tmsg = strings.TrimSpace(msg)")
	stats = strings.TrimSpace(stats)
	log.Println("\tcommandArr := strings.Fields(comStr)")
	statsArr := strings.Fields(stats)
	log.Println("\tfileView.history.Append(tui.NewLabel(comStr))")

	//fileView.history.Append(tui.NewLabel(stats))
	log.Println("buffer := make([]byte, 1024)")
	log.Println("len = ", len(statsArr))
	log.Println(statsArr)
	if len(statsArr) < 3 {
		in2 <- "| Файл не существует или ошибка соединения"
		return
	}
	log.Println(statsArr[2])
	fileSize, err := strconv.ParseInt(statsArr[2], 10, 64)
	if err != nil || fileSize == -1 {
		log.Println("file size error", err)
		in2 <- "| Ошибка размера файла " + err.Error()
		return
	}
	//fileView.history.Append(tui.NewLabel(fmt.Sprintf("/startdsend %s\n", strings.TrimPrefix(msg, "/download "))))
	conn.Write([]byte(fmt.Sprintf("/startdsend %s\n", strings.TrimPrefix(msg, "/download "))))

	buf := new(bytes.Buffer)
	io.Copy(buf, io.LimitReader(conn2, fileSize))

	outputFile, err := os.Create(rootDownload + "/" + strings.TrimPrefix(msg, "/download "))
	if err != nil {
		log.Println(err)
	}
	io.Copy(outputFile, bytes.NewReader(buf.Bytes()))
	defer outputFile.Close()

	// conn.Write([]byte("File Downloaded successfully"))
	in2 <- fmt.Sprintf("| Файл %v скачан", strings.TrimPrefix(msg, "/download"))
	log.Println("File Downloaded successfully")

	//checkFileMD5Hash(ROOT + "/" + fname)
}

func sendFile(conn net.Conn, conn2 net.Conn, msg string) {

	f, err := os.OpenFile("testlogfile", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()

	log.SetOutput(f)
	args := strings.Fields(msg)
	log.Println("Openin file", args[1])
	data, err := ioutil.ReadFile(rootUpload + "/" + args[1])
	if err != nil {
		in2 <- "| Ошибка открытия файла " + err.Error()
		log.Println(err)
		return
	}
	log.Println("Conn write", msg, len(data))
	conn.Write([]byte(fmt.Sprintf("%s %d\n", msg, len(data))))

	log.Println("\treader := bufio.NewReader(conn)")
	reader := bufio.NewReader(conn2)
	log.Println("\tmsg, err := reader.ReadString('\\n')")
	incoming, err := reader.ReadString('\n')
	log.Println("\tmsg = strings.TrimSpace(msg)")
	incoming = strings.TrimSpace(incoming)
	log.Println("\tcommandArr := strings.Fields(comStr)")
	statsArr := strings.Fields(incoming)
	if statsArr[0] != "200" {
		in2 <- "| Ошибка при загрузке " + incoming
		log.Println(incoming)
		return
	}

	io.Copy(conn2, bytes.NewReader(data))
	incoming, err = reader.ReadString('\n')
	in2 <- incoming
	// checkFileMD5Hash(ROOT + "/" + fname)
}

func listFiles(conn net.Conn, conn2 net.Conn, msg string) {
	f, err := os.OpenFile("testlogfile", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)
	conn.Write([]byte(fmt.Sprintf(msg + "\n")))

	log.Println("\treader := bufio.NewReader(conn)")
	reader := bufio.NewReader(conn2)
	log.Println("\tmsg, err := reader.ReadString('\\n')")
	list, err := reader.ReadString('|')
	log.Println("\tmsg = strings.TrimSpace(msg)")
	list = strings.TrimSpace(list)
	log.Println("\tcommandArr := strings.Fields(comStr)")

	files := strings.Fields(list)
	log.Println(files)
	log.Println("\tfileView.history.Append(tui.NewLabel(comStr))")
	in2 <- list[:len(list)-1]
}
