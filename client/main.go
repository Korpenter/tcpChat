package main

import (
	"bufio"
	"fmt"
	"github.com/marcusolsson/tui-go"
	"io"
	"log"
	"net"
	"strings"
)

// подключение по TCP, по заданному порту
func main() {
	var port int
	var conn net.Conn
	var err error
	for { // пока не задан порт на котором работает сервер, чтение порта
		fmt.Println("Введите порт для подключения: ")
		fmt.Scanln(&port)                                          // ввод порта
		conn, err = net.Dial("tcp", "localhost:"+fmt.Sprint(port)) // подлкючение по TCP сокету
		if err != nil {
			fmt.Println(err)
		} else {
			break
		}
	}
	var loggedIn bool               // флаг авторизации
	chatView := newChatView(conn)   // окно чата
	loginView := newLoginView(conn) // окно авторизации
	ui, err := tui.New(loginView)   // инициализация ui

	ui.SetKeybinding("Esc", func() { ui.Quit() }) // Esc - выход из приложения
	// стрелка вверх - пролистать чат вверх
	ui.SetKeybinding("Up", func() { chatView.historyScroll.Scroll(0, -1) })
	// стрелка вниз - пролистать чат вниз
	ui.SetKeybinding("Down", func() { chatView.historyScroll.Scroll(0, 1) })
	// стрелка вправо - пролистать до конца
	ui.SetKeybinding("Right", func() { chatView.historyScroll.ScrollToBottom() })
	// стрелка влево - пролистать до начала
	ui.SetKeybinding("Left", func() { chatView.historyScroll.ScrollToTop() })

	if err != nil {
		fmt.Println(err)
	}

	defer conn.Close() // закытие соединения при окончании выполнения

	go func() { // запуск рутины функции для чтения из соединения
		for {
			reader := bufio.NewReader(conn)
			msg, err := reader.ReadString('\n')
			msg = strings.TrimSpace(msg)
		l:
			switch err { // если ошибка сервера, вывод ошибки
			case nil:
				if !loggedIn { // если не авторизован рендерится окно авторизации
					switch msg {
					case "успешная регистрация", "успешный вход": // если успешная авторизация
						ui.SetWidget(chatView) // смена окна на чат
						loggedIn = true        // установка фалага авторизации
						break l
					default: // если безуспешная авторизация - выводится статус ошибки
						ui.Update(func() {
							loginView.status.SetText(msg)
						})
						break l
					}
				}
				// обновляется список сообщений при поступленгии от сервера
				ui.Update(func() {
					chatView.history.Append(tui.NewHBox(
						tui.NewLabel(fmt.Sprintf("%v", msg)),
					))
				})
			case io.EOF:
				log.Println("Сервер закрыл соединение")
				return
			default:
				log.Printf("Ошибка сервера: %v\n", err)
				return
			}

		}
	}()
	// запуск UI
	if err := ui.Run(); err != nil {
		panic(err)
	}
}
