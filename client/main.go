package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/marcusolsson/tui-go"
	"io"
	"log"
	"net"
	"os"
	"strings"
)

const rootUpload = "upload"       // папка в которую помещаются файлы для загрузки на сервер
const rootDownload = "downloaded" // папка в которую помещаются скачанные файлы

var port int                // порт, если не введен, 755
var addr string             // а
var in1 = make(chan string) // канал для обновления интерфейса чата
var in2 = make(chan string) // канал для обновления интерфейса файлов
var f *os.File

// подключение по TCP, по заданному порту

func main() {
	flag.IntVar(&port, "port", 755, "Порт") // флаг порта
	var conn net.Conn                       // tcp соединение
	var conn2 net.Conn                      // tcp соединение
	var err error
	if _, err := os.Stat("clientLogs.txt"); os.IsNotExist(err) {
		f, err = os.OpenFile("clientLogs.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666) // открытие для логов
		if err != nil {
			log.Fatalf("error opening file: %v", err)
		}
	}
	defer f.Close()
	log.SetOutput(f)

	if _, err := os.Stat(rootUpload); os.IsNotExist(err) { // создание папки для загрузок на сервер, если не создана
		if err = os.Mkdir(rootUpload, 0777); err != nil {
			log.Println(err)
			return
		}
	}
	if _, err := os.Stat(rootDownload); os.IsNotExist(err) { // создание папки для загрузок с сервера, если не создана
		if err = os.Mkdir(rootDownload, 0777); err != nil {
			log.Println(err)
			return
		}
	}

	out := make(chan []byte)                                   // канал на отправку по соединению conn1
	log.Println("running on port:", port)                      //, *port2)
	conn, err = net.Dial("tcp", "localhost:"+fmt.Sprint(port)) // подлкючение по TCP сокету
	if err != nil {
		log.Println(err)
	}
	conn2, err = net.Dial("tcp", "localhost:"+fmt.Sprint(port)) // подлкючение по TCP сокету для файлов
	if err != nil {
		log.Println(err)
	}

	var loggedIn bool              // флаг авторизации
	chatView := newChatView(out)   // окно чата
	loginView := newLoginView(out) // окно авторизации
	fileView := newFileView(conn, conn2)
	currentView := 0
	ui, err := tui.New(loginView)

	ui.SetKeybinding("Esc", func() { ui.Quit() }) // Esc - выход из приложения
	// стрелка вверх - пролистать чат вверх
	ui.SetKeybinding("Up", func() {
		if currentView == 1 { // зависит от того, какое окно активно
			chatView.historyScroll.Scroll(0, -1)
		}
		if currentView == 2 {
			fileView.historyScroll.Scroll(0, -1)
		}
	})
	// стрелка вниз - пролистать чат вниз
	ui.SetKeybinding("Down", func() {
		if currentView == 1 {
			chatView.historyScroll.Scroll(0, 1)
		}
		if currentView == 2 {
			fileView.historyScroll.Scroll(0, 1)
		}
	})
	// стрелка вправо - пролистать до конца
	ui.SetKeybinding("Right", func() {
		if currentView == 1 {
			chatView.historyScroll.ScrollToBottom()
		}
		if currentView == 2 {
			fileView.historyScroll.ScrollToBottom()
		}
	})
	// стрелка влево - пролистать до начала
	ui.SetKeybinding("Left", func() {
		if currentView == 1 {
			chatView.historyScroll.ScrollToTop()
		}
		if currentView == 2 {
			fileView.historyScroll.ScrollToBottom()
		}
	})

	if err != nil {
		log.Println(err)
	}

	defer conn.Close() // закытие соединения при окончании выполнения
	defer conn2.Close()

	go func() { // запуск рутины функции для чтения из соединения
		for {
			reader := bufio.NewReader(conn)     // объект для чтения из соединения
			msg, err := reader.ReadString('\n') // чтение строки до символа \n
			msg = strings.TrimSpace(msg)
			log.Println("msg", msg)

		l:
			switch err { // если ошибка сервера, вывод ошибки
			case nil:
				if !loggedIn { // если не авторизован рендерится окно авторизации
					switch {
					case strings.HasPrefix(msg, "50"):
						conn.Write([]byte("100\n"))
						addr = strings.TrimPrefix(msg, "50")
						break l
					case strings.HasPrefix(msg, "201"), strings.HasPrefix(msg, "202"): // если успешная авторизация
						ui.Update(func() { ui.SetWidget(chatView) }) // смена окна на чат
						currentView = 1
						ui.SetKeybinding("Tab", func() { // бинд на смену окон на Tab
							if currentView == 1 {
								currentView = 2
								ui.SetWidget(fileView)
							} else {
								currentView = 1
								ui.SetWidget(chatView)
							}
						})
						loggedIn = true // установка фалага авторизации
						break l
					default: // если безуспешная авторизация - выводится статус ошибки
						ui.Update(func() {
							loginView.status.SetText(msg)
						})
						break l
					}
				}

				in1 <- msg // отправка сообщения в канал, для обновления интерфейса
			case io.EOF:
				in1 <- "| Сервер закрыл соединение"
				in2 <- "| Сервер закрыл сединение"
				return
			default:
				log.Printf("server error: %v\n", err)
				return
			}

		}
	}()

	go write(conn, out) // запуск рутины для записи в соединение

	reader := bufio.NewReader(conn2) // объект для чтения из второго соединения
	msg, err := reader.ReadString('\n')
	msg = strings.TrimSpace(msg)

	if strings.HasPrefix(msg, "50") { // проверка подтверждения на подключение
		conn2.Write([]byte("105 " + addr + "\n"))
	}

	go func() { // запуск рутины функции для чтения из соединения
		for { //
			select {
			case val, ok := <-in1: // обновление интерфейса
				if !ok {
					return
				}
				ui.Update(func() {
					if strings.HasPrefix(val, "|") {
						chatView.status.SetText(val)
						return
					}
					rows := strings.Split(val, "\n") // добавление файлов в список
					for _, row := range rows {
						chatView.table.AppendRow(tui.NewLabel(row))
						chatView.historyScroll.Scroll(0, 1)
					}
				})
			case val, ok := <-in2: // обновление интерфейса
				if !ok {
					return
				}
				ui.Update(func() {
					if strings.HasPrefix(val, "|") {
						fileView.status.SetText(val)
						return
					}
					rows := strings.Split(val, "\n") // добавление файлов в список
					for _, row := range rows {
						fileView.table.AppendRow(tui.NewLabel(row))
					}
				})
			}
		}
	}()
	log.Println("init ui")
	if err := ui.Run(); err != nil {
		panic(err)
	} // запуск UI
}

func write(conn net.Conn, out chan []byte) { // функция записи в соединение
	for {
		select {
		case msg, ok := <-out: // отправка сообщения из канала
			if !ok {
				return
			}
			conn.Write(msg) // запись в соединение
		}
	}
}
