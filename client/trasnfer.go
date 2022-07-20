package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

func getFile(conn net.Conn, fname string) {

	conn.Write([]byte(fmt.Sprintf("/download %s\n", fname)))

	f, err := os.OpenFile("testlogfile.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()

	log.SetOutput(f)

	buffer := make([]byte, 1024)
	n, _ := conn.Read(buffer)
	comStr := strings.Trim(string(buffer[:n]), "\n")
	commandArr := strings.Fields(comStr)

	fileSize, err := strconv.ParseInt(commandArr[2], 10, 64)
	if err != nil || fileSize == -1 {
		log.Println("file size error", err)
		conn.Write([]byte("file size error\n"))
		return
	}

	conn.Write([]byte("200 Start download!\n"))

	buf := new(bytes.Buffer)
	io.Copy(buf, io.LimitReader(conn, fileSize))

	//	arrDec, err := CBCDecrypter(myFPass, buf.Bytes())
	//	if err != nil {
	//		log.Println(err)
	//		return
	//	}

	outputFile, err := os.Create(rootDownload + "/" + fname)
	if err != nil {
		log.Println(err)
	}
	io.Copy(outputFile, bytes.NewReader(buf.Bytes()))
	defer outputFile.Close()

	// conn.Write([]byte("File Downloaded successfully"))
	log.Println("File Downloaded successfully")

	//checkFileMD5Hash(ROOT + "/" + fname)
}
