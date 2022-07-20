package tcpChatServer

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

// модель сервера
type server struct {
	rooms    map[string]*room     // активные комнаты
	commands chan command         // канал поступающих команд
	online   map[net.Conn]*client // пользователи в сети
	tempmap  map[net.Conn]*client
}

// newServer возвращает новый сервер
func newServer() *server {
	return &server{
		rooms:    make(map[string]*room),
		commands: make(chan command),
		online:   make(map[net.Conn]*client),
		tempmap:  make(map[net.Conn]*client),
	}
}

// start запускает вызывает рутину запуска сервера
func Start(port int) {
	go func() { serverStart(port) }()
}

// serverStart запускает сервер на заданном порте
func serverStart(port int) {
	s := newServer()
	go s.run() // запуск рутин обработки команд
	fmt.Println("Запуск сервера на порте: " + fmt.Sprint(port))
	// подключение к бд
	db := getdb("")
	listener, err := net.Listen("tcp", ":"+fmt.Sprint(port)) // запуск на TCP socket для приема входящих соединений
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Сервер запущен на порте: " + fmt.Sprint(port))
	//fmt.Println("Запуск сервера на порте: " + fmt.Sprint(port+1))
	//listener2, err := net.Listen("tcp", ":"+fmt.Sprint(port+1)) // запуск на TCP socket для приема входящих соединений
	//if err != nil {
	//	fmt.Println(err)
	//	return
	//}
	defer listener.Close()
	//defer listener2.Close()
	//fmt.Println("Сервер запущен на порте: " + fmt.Sprint(port+1))
	s.acceptLoop(listener, db) // для каждого подключающегося клиента создается модель newClient
	//s.acceptLoop(listener2, db) // run in the main goroutine
}

func (s *server) acceptLoop(l net.Listener, db *mongo.Client) {
	defer l.Close()
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("New connection found!")
		s.regConn(conn, db)
	}
}

func (s *server) regConn(conn net.Conn, db *mongo.Client) {
	conn.Write([]byte("50\n"))
	for {
		msg, err := bufio.NewReader(conn).ReadString('\n') // чтение из соединения
		if err != nil {
			fmt.Println(err)
			return
		}
		msg = strings.Trim(msg, "\r\n")
		fmt.Println("msg = ", msg)
		args := strings.Split(msg, " ")
		fmt.Println("args = ", args)
		code := strings.TrimSpace(args[0])
		fmt.Println("code = ", args[0])
		switch code {
		case "100":
			fmt.Println("100")
			s.newClient(conn, db)
			fmt.Println("101")
			return
		case "105":
			fmt.Println("105")
			fmt.Println("addr = ", args[1])
			for _, c := range s.tempmap {
				fmt.Println("c.chatConn.RemoteAddr().String() = ", c.chatConn.RemoteAddr().String())
				if args[1] == c.chatConn.RemoteAddr().String() {
					c.fileConn = conn
					fmt.Println("Зарегистрировано второе соединение")
					return
				}
			}
		}
	}
}

// обработка команд
func (s *server) run() {
	// для каждой команды определяется тип и вызывается функция обработчик
	for cmd := range s.commands {
		switch cmd.id {
		case CMD_LOGIN:
			s.login(cmd.client, cmd.args, cmd.db)
		case CMD_SIGNUP:
			s.signUP(cmd.client, cmd.args, cmd.db)
		case CMD_JOIN:
			s.join(cmd.client, cmd.args)
		case CMD_ROOMS:
			s.listRooms(cmd.client)
		case CMD_MSG:
			s.msg(cmd.client, cmd.args)
		case CMD_DOWNLOAD:
			s.sendFile(cmd.client, cmd.args)
		case CMD_QUIT:
			s.quit(cmd.client)
		}
	}
}

// newClient функция возвращает модель клиента
func (s *server) newClient(conn net.Conn, db *mongo.Client) {
	log.Printf("Новый пользователь: %s", conn.RemoteAddr().String())

	c := &client{
		chatConn: conn,
		commands: s.commands,
	}

	c.msg("Address:" + conn.RemoteAddr().String())
	s.tempmap[c.chatConn] = c
	go c.readInput(db)
}

// login функция отвечает за авторизацию пользователя
func (s *server) login(c *client, args []string, db *mongo.Client) {
	//контекст -  если операция не выполнится за 80 секунд - отмена
	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Second)
	defer cancel()
	fmt.Println("Новая попытка входа: ", args)
	fmt.Println(s.online)
	// проверка авторизована ли эта учетная запись, и отключение авторизованног клиента в этом случае
	for _, v := range s.online {
		fmt.Println(args[1], v)
		if args[1] == v.username {
			v.msg("Учетная запись исползована на другом пк, отключение")
			v.chatConn.Close()
			// удаление авторизованного клиента и списка онлайн пользователей
			delete(s.online, v.chatConn)
			break
		}
	}

	usersDB := db.Database("users") // подключение к бд
	userCollection := usersDB.Collection("users")

	if len(args) != 3 { // если не предоставлены логи и пароль - ошибка
		c.err(errors.New("введите логин и пароль"))
		return
	}

	user := userCollection.FindOne(ctx, bson.M{"username": args[1]}) // поиск по БД по имени пользователя
	var userField bson.M
	err := user.Err()
	if err != nil {
		fmt.Println(err.Error())
		c.err(errors.New("пользователь не найден"))
		return
	}

	err = user.Decode(&userField) // раскодирование из BSON
	if err != nil {
		c.err(errors.New("ошибка базы данных" + err.Error() + "\n"))
		return
	}

	if userField["password"] == args[2] { // проверка пароля
		c.msg("успешный вход")
		c.loggedIn = true
		c.username = args[1]
		delete(s.tempmap, c.chatConn)
		s.online[c.chatConn] = c
		return
	} else {
		c.err(errors.New("неверный пароль"))
		return
	}
}

// signUP отвечает за авториза
func (s *server) signUP(c *client, args []string, db *mongo.Client) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if len(args) != 3 {
		c.err(errors.New("введите логин и пароль"))
		return
	}
	fmt.Println("новая попытка регистрации: ", args)
	usersDb := db.Database("users")
	userCollection := usersDb.Collection("users")

	hasUsername := userCollection.FindOne(ctx, bson.M{ // проверка существует ли пользователь
		"username": args[1],
	})

	if hasUsername.Err() == nil { // если имя занято
		c.err(errors.New("имя пользователя уже использовано"))
		return
	}

	_, err := userCollection.InsertOne(ctx, bson.D{ // добавлени пользователя в бд
		{Key: "username", Value: args[1]},
		{Key: "password", Value: args[2]},
	})
	if err != nil {
		c.err(errors.New("ошибка базы данных"))
		return
	}
	c.msg(fmt.Sprintf("успешная регистрация"))
	c.username = args[1]     // установка имени пользователя
	s.online[c.chatConn] = c // добавление в онлайн список
	c.loggedIn = true
}

// join отвечает за вход в комнату
func (s *server) join(c *client, args []string) {
	if len(args) != 2 {
		c.err(errors.New("неверный формат команды"))
		return
	}
	roomName := args[1]
	r, ok := s.rooms[roomName] // если комната не существует создаем и добавляем в списсок
	if !ok {
		r = &room{
			name:    roomName,
			members: make(map[net.Addr]*client),
		}
		s.rooms[roomName] = r
	}

	r.members[c.chatConn.RemoteAddr()] = c // добавление пользователя в список активных в комнате
	s.quitCurrentRoom(c)                   // отключение от нынешней комнаты
	c.room = r                             // установка комнаты пользователя
	r.broadcast(fmt.Sprintf("!!! %s вошел в комнату %s", c.username, r.name))
	c.msg(fmt.Sprintf("добро пожаловать в %s", r.name))
}

// listRooms возвращает список активных комнат
func (s *server) listRooms(c *client) {
	var rooms []string
	for name := range s.rooms {
		rooms = append(rooms, name)
	}

	c.msg(fmt.Sprintf("!!! доступные комнаты: %s", strings.Join(rooms, ", ")))
}

// msg отправляет сообщение всем пользователя в комнате
func (s *server) msg(c *client, args []string) {
	if c.room == nil {
		c.err(errors.New("!!! сперва войдите в комнату"))
		return
	}

	c.room.broadcast(c.username + ": " + strings.Join(args[1:len(args)], " "))
}

// quit отвечает за отключение от сервера и закрытие соединения
func (s *server) quit(c *client) {
	log.Printf("!!! клиент отключился: %s", c.chatConn.RemoteAddr().String())

	s.quitCurrentRoom(c)         // выход из комнаты
	for _, v := range s.online { // удаление из списка онлайн пользователей
		if v == c {
			delete(s.online, c.chatConn)
		}
	}
	c.msg("!!! отключение от сервера")
	c.chatConn.Close() // закрытие соединения
}

// quitCurrentRoom отвечает за выход из комнаты
func (s *server) quitCurrentRoom(c *client) {
	if c.room != nil { // если пользователь в комнате
		delete(c.room.members, c.chatConn.RemoteAddr()) // удаляем пользователя из списка активных в комнате
		if len(c.room.members) == 0 {                   // если комната пустая - она удаляется
			delete(s.rooms, c.room.name) // удаление комнаты
		} else {
			c.room.broadcast(fmt.Sprintf("%s вышел из комнаты", c.username))
		}
	}
}

func (s *server) sendFile(c *client, args []string) {
	inputFile, err := os.Open(args[1])
	if err != nil {
		log.Println(err.Error())
		c.fileConn.Write([]byte(err.Error()))
		return
	}
	defer inputFile.Close()

	stats, _ := inputFile.Stat()

	c.fileConn.Write([]byte(fmt.Sprintf("download %s %d\n", args[1], stats.Size())))

	buf := make([]byte, 1024)
	n, err := c.fileConn.Read(buf) //TODO timeout wating
	if err != nil {
		log.Println(err.Error())
		return
	}

	str := strings.Trim(string(buf[:n]), "\n")
	commandArr := strings.Fields(str)
	if commandArr[0] != "200" {
		log.Println(str)
		return
	}

	io.Copy(c.fileConn, inputFile)

	log.Println("File ", args[1], " Send successfully")
	c.msg("file" + args[1] + "send successfully")
}
