package main

import (
	"fmt"
	"github.com/marcusolsson/tui-go"
	"log"
	"net"
	"strings"
)

// лого на окне авторизации
var logo = `       		   			           __           __   
  __________   ____ |  | __ _____/  |_ 
 /  ___/  _ \_/ ___\|  |/ // __ \   __\
 \___ (  <_> )  \___|    <\  ___/|  |  
/____  >____/ \___  >__|_ \\___  >__|  
     \/           \/     \/    \/     `

type boxView struct { // модель окна чата и файлов
	tui.Box
	history       *tui.Box
	historyScroll *tui.ScrollArea
	layout        *tui.Box
	sidebar       *tui.Box
}

type loginView struct { // модель окна авторизации
	tui.Box
	layout *tui.Box
	status *tui.StatusBar
}

// newChatView возвращает новое окно чата
func newChatView(out chan []byte) *boxView {
	view := &boxView{}
	sidebar := tui.NewVBox( // боковая панель с подксказками
		tui.NewLabel("Команды"),
		tui.NewLabel("доступные комнаты:\n/rooms"),
		tui.NewLabel("войти в комнату:\n/join"),
		tui.NewLabel("выйти:\n/quit"),
		tui.NewLabel("сообщение:\nтекст без '/'"),
		tui.NewLabel("перейти в окно файлов:\n/files"),
		tui.NewLabel("\nПеремещение"),
		tui.NewLabel("пролистать чат вверх:\nUpArrow"),
		tui.NewLabel("пролистать чат вниз:\nDownArrow"),
		tui.NewLabel("первое сообщение:\nLeftArrow"),
		tui.NewLabel("последнее сообщение:\nRightArrow"),
		tui.NewSpacer(),
		tui.NewLabel("\nTab - переход в окно файлов"),
	)
	sidebar.SetBorder(true) // видимая граница панели
	view.history = tui.NewVBox()

	view.historyScroll = tui.NewScrollArea(view.history) // добавление возможности прокрутки сообщений
	view.historyScroll.ScrollToBottom()

	historyBox := tui.NewVBox(view.historyScroll)
	historyBox.SetBorder(true)

	input := tui.NewEntry() // добавление окна для ввода команд, сообщений
	input.SetFocused(true)
	input.SetSizePolicy(tui.Expanding, tui.Maximum)

	inputBox := tui.NewHBox(input)
	inputBox.SetBorder(true)
	inputBox.SetSizePolicy(tui.Expanding, tui.Maximum)

	chat := tui.NewVBox(historyBox, inputBox) // компоновка элементов в вертикальном формате
	chat.SetSizePolicy(tui.Expanding, tui.Expanding)

	input.OnSubmit(func(e *tui.Entry) {
		if e.Text() != "" { // если ввод не пуст
			msg := strings.TrimSpace(e.Text()) // удаление пробелов
			if !strings.HasPrefix(msg, "/") {  // если нет слеша, то отправляется, как обычное сообщение
				msg = "/msg " + msg
			}
			out <- []byte(msg + "\n")           // запись в канал
			view.historyScroll.ScrollToBottom() // прокрутка вниз при отправке сообщения
			e.SetText("")                       // сброс ввода
		}
	})

	view.layout = tui.NewHBox( // компановка в горизонтальном формате
		chat,
		sidebar,
	)

	view.layout.SetBorder(false) // установка границы всего окна
	view.Append(view.layout)

	return view
}

// newFileView возвращает новое окно для файлов
func newFileView(out2 chan []byte) *boxView {
	view := &boxView{}
	sidebar := tui.NewVBox( // боковая панель с подксказками
		tui.NewLabel("Команды"),

		tui.NewLabel("просмотр общих файлов:\nlist"),
		tui.NewLabel("скачать общий файл:\ndownload <filename>"),
		tui.NewLabel("загрузить общий:\nupload <filename>"),
		tui.NewLabel("\nПеремещение"),
		tui.NewLabel("пролистать окно файлов вверх:\nUpArrow"),
		tui.NewLabel("пролистать окно файлов вниз:\nDownArrow"),
		tui.NewLabel("первое файл:\nLeftArrow"),
		tui.NewLabel("последнее файл:\nRightArrow"),
		tui.NewSpacer(),
		tui.NewLabel("Tab - переход в окно чата"),
	)
	sidebar.SetBorder(true) // видимая граница панели
	view.history = tui.NewVBox()

	view.historyScroll = tui.NewScrollArea(view.history) // добавление возможности прокрутки файлов
	view.historyScroll.ScrollToBottom()

	historyBox := tui.NewVBox(view.historyScroll)
	historyBox.SetBorder(true)

	input := tui.NewEntry() // добавление окна для ввода команд, сообщений
	input.SetFocused(true)
	input.SetSizePolicy(tui.Expanding, tui.Maximum)

	inputBox := tui.NewHBox(input)
	inputBox.SetBorder(true)
	inputBox.SetSizePolicy(tui.Expanding, tui.Maximum)

	chat := tui.NewVBox(historyBox, inputBox) // компоновка элементов в вертикальном формате
	chat.SetSizePolicy(tui.Expanding, tui.Expanding)
	input.OnSubmit(func(e *tui.Entry) {
		if e.Text() != "" { // если ввод не пуст
			//msg := strings.TrimSpace(e.Text())  // удаление пробелов
			//getFile(conn2, msg)                 // запись в канал
			view.historyScroll.ScrollToBottom() // прокрутка вниз при отправке сообщения
			e.SetText("")                       // сброс ввода
		}
	})

	view.layout = tui.NewHBox( // компановка в горизонтальном формате
		chat,
		sidebar,
	)

	view.layout.SetBorder(false) // установка границы всего окна
	view.Append(view.layout)

	return view
}

// newLoginView возвращает новое окно авторизации
func newLoginView(conn net.Conn) *loginView {
	view := &loginView{}

	user := tui.NewEntry() // поле ввода для имени пользователя
	user.SetFocused(true)  // курсор на нем

	password := tui.NewEntry() // поле ввода для пароля
	password.SetEchoMode(tui.EchoModePassword)

	form := tui.NewGrid(0, 0) // поле для названий логина и пароля
	form.AppendRow(tui.NewLabel("Логин"), tui.NewLabel("Пароль"))
	form.AppendRow(user, password)

	view.status = tui.NewStatusBar("Ожидание ввода.") // статус авторизации

	login := tui.NewButton("[Вход]") // кнопка для авторизации

	login.OnActivated(func(b *tui.Button) { // при нажатии копки авторизации
		if user.Text() != "" && password.Text() != "" { // если поля пользователя и пароля не пусты
			user := strings.TrimSpace(user.Text()) // удаление пробелов
			password := strings.TrimSpace(password.Text())
			cmd := fmt.Sprintf("/login %v %v", user, password) // форматирование команды
			_, err := conn.Write([]byte(cmd + "\n"))           // отправка команды
			if err != nil {
				log.Printf("write text `%s` failed with err: %s\n", cmd, err.Error())
			}
		} else {
			view.status.SetText("Введите логин и пароль!")
		}
	})

	register := tui.NewButton("[Регистрация]") // кнопка для регистрации

	register.OnActivated(func(b *tui.Button) {
		if user.Text() != "" && password.Text() != "" {
			user := strings.TrimSpace(user.Text())
			password := strings.TrimSpace(password.Text())
			cmd := fmt.Sprintf("/signup %v %v", user, password)
			_, err := conn.Write([]byte(cmd + "\n"))
			if err != nil {
				log.Printf("ошибка `%s` err: %s\n", cmd, err.Error())
			}
		} else {
			view.status.SetText("Введите логин и пароль!")
		}
	})

	buttons := tui.NewHBox( // компановка кнопок в горизонтальном формате
		tui.NewSpacer(),
		tui.NewPadder(1, 0, login),
		tui.NewPadder(1, 0, register),
	)

	window := tui.NewVBox( // компановка кнопок в вертикальном формате
		tui.NewPadder(10, 1, tui.NewLabel(logo)),
		tui.NewPadder(12, 0, tui.NewLabel("Войдите и зарегистрируйтесь.")),
		tui.NewPadder(1, 1, form),
		buttons,
	)
	window.SetBorder(true) // видимая граница окна

	wrapper := tui.NewVBox(
		tui.NewSpacer(),
		window,
		tui.NewSpacer(),
	)

	content := tui.NewHBox(tui.NewSpacer(), wrapper, tui.NewSpacer())

	layout := tui.NewVBox(
		content,
		view.status,
	)

	view.layout = layout

	view.layout.SetBorder(false)
	view.Append(view.layout)

	tui.DefaultFocusChain.Set(user, password, login, register) // в каком порядке TAB переходит по элементам окна

	return view
}
