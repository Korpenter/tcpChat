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
	"time"
)

const rootUpload = "upload"
const rootDownload = "downloaded"

var port1 = flag.Int("port1", 8888, "Порт1")
var port2 = flag.Int("port2", 8889, "Порт2")
var addr string
var in1 = make(chan string)
var in2 = make(chan string)

// подключение по TCP, по заданному порту
func main() {

	if _, err := os.Stat(rootUpload); os.IsNotExist(err) {
		if err = os.Mkdir(rootUpload, 0777); err != nil {
			fmt.Println(err)
			time.Sleep(time.Second * 3)
			return
		}
	}
	if _, err := os.Stat(rootDownload); os.IsNotExist(err) {
		if err = os.Mkdir(rootDownload, 0777); err != nil {
			fmt.Println(err)
			time.Sleep(time.Second * 3)
			return
		}
	}

	var err error
	var conn net.Conn  // tcp соединение
	var conn2 net.Conn // tcp соединение
	out := make(chan []byte)
	fmt.Println("Запуск на порте:", *port1)                      //, *port2)
	conn, err = net.Dial("tcp", "localhost:"+fmt.Sprint(*port1)) // подлкючение по TCP сокету
	if err != nil {
		fmt.Println(err)
	}
	conn2, err = net.Dial("tcp", "localhost:"+fmt.Sprint(*port1)) // подлкючение по TCP сокету для файлов
	if err != nil {
		fmt.Println(err)
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
		if currentView == 1 {
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
		fmt.Println(err)
	}

	defer conn.Close() // закытие соединения при окончании выполнения
	defer conn2.Close()

	go func() { // запуск рутины функции для чтения из соединения
		for {
			reader := bufio.NewReader(conn)
			msg, err := reader.ReadString('\n')
			msg = strings.TrimSpace(msg)
			fmt.Println("msg", msg)

		l:
			switch err { // если ошибка сервера, вывод ошибки
			case nil:
				if !loggedIn { // если не авторизован рендерится окно авторизации
					switch {
					case strings.HasPrefix(msg, "50"):
						conn.Write([]byte("100\n"))
						addr = strings.TrimPrefix(msg, "50")
					case strings.HasPrefix(msg, "201"), strings.HasPrefix(msg, "202"): // если успешная авторизация
						ui.Update(func() { ui.SetWidget(chatView) }) // смена окна на чат
						currentView = 1
						ui.SetKeybinding("Tab", func() {
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
				// обновляется список сообщений при поступленгии от сервера
				//		if !strings.HasPrefix(msg, ">") || currentView == 1 && strings.HasPrefix(msg, "ERR:") {
				//			ui.Update(func() {
				//				fileView.history.Append(tui.NewHBox(
				//					tui.NewLabel(fmt.Sprintf("%v", msg)),
				//				))
				//				fileView.historyScroll.Scroll(0, 1)
				//			})
				//		}
				in1 <- msg
			case io.EOF:
				in1 <- "| Сервер закрыл соединение"
				in2 <- "| Сервер закрыл сединение"
				return
			default:
				log.Printf("Ошибка сервера: %v\n", err)
				return
			}

		}
	}()

	go write(conn, out)
	// запуск UI

	reader := bufio.NewReader(conn2)
	msg, err := reader.ReadString('\n')
	msg = strings.TrimSpace(msg)

	if strings.HasPrefix(msg, "50") {
		conn2.Write([]byte("105 " + addr + "\n"))
	}

	go func() {
		for {
			select {
			case val, ok := <-in1:
				if !ok {
					return
				}
				ui.Update(func() {
					if strings.HasPrefix(val, "|") {
						chatView.status.SetText(val)
						return
					}
					label := tui.NewLabel(val)
					label.SetWordWrap(true)
					chatView.history.Append(label)
					chatView.historyScroll.Scroll(0, 1)
				})
			case val, ok := <-in2:
				if !ok {
					return
				}
				ui.Update(func() {
					if strings.HasPrefix(val, "|") {
						fileView.status.SetText(val)
						return
					}
					rows := strings.Fields(val)
					for _, row := range rows {
						fileView.table.AppendRow(tui.NewLabel(row))
					}
				})
			}
		}
	}()

	if err := ui.Run(); err != nil {
		panic(err)
	} // инициализация ui

	//go func() { // запуск рутины функции для чтения из соединения
	//	for {
	//		reader := bufio.NewReader(conn2)
	//		msg, err := reader.ReadString('\n')
	//		msg = strings.TrimSpace(msg)
	//	l:
	//		switch err { // если ошибка сервера, вывод ошибки
	//		case nil:
	//			if !loggedIn { // если не авторизован рендерится окно авторизации
	//				switch {
	//				case msg == "50":
	//					ui.Update(func() {
	//						loginView.status.SetText("второе msg")
	//					})
	//					conn2.Write([]byte("105 " + addr + "\n"))
	//					break l
	//				case msg == "> успешная регистрация", msg == "> успешный вход": // если успешная авторизация
	//					ui.Update(func() { ui.SetWidget(chatView) }) // смена окна на чат
	//					currentView = 1
	//					ui.SetKeybinding("Tab", func() {
	//						if currentView == 1 {
	//							currentView = 2
	//							ui.SetWidget(fileView)
	//						} else {
	//							currentView = 1
	//							ui.SetWidget(chatView)
	//						}
	//					})
	//					loggedIn = true // установка фалага авторизации
	//					break l
	//				default: // если безуспешная авторизация - выводится статус ошибки
	//					ui.Update(func() {
	//						loginView.status.SetText(msg)
	//					})
	//					break l
	//				}
	//			}
	//			// обновляется список сообщений при поступленгии от сервера
	//			if !strings.HasPrefix(msg, ">") || currentView == 1 && strings.HasPrefix(msg, "ERR:") {
	//				ui.Update(func() {
	//					fileView.history.Append(tui.NewHBox(
	//						tui.NewLabel(fmt.Sprintf("%v", msg)),
	//					))
	//					fileView.historyScroll.Scroll(0, 1)
	//				})
	//			}
	//			ui.Update(func() {
	//				chatView.history.Append(tui.NewHBox(
	//					tui.NewLabel(fmt.Sprintf("%v", msg)),
	//				))
	//				chatView.historyScroll.Scroll(0, 1)
	//			})
	//		case io.EOF:
	//			log.Println("Сервер закрыл соединение")
	//			return
	//		default:
	//			log.Printf("Ошибка сервера: %v\n", err)
	//			return
	//		}
	//
	//	}
	//}()

}

func write(conn net.Conn, out chan []byte) {
	for {
		select {
		case msg, ok := <-out:
			if !ok {
				return
			}
			conn.Write(msg)
		}
	}
}
