package tcpServer

import (
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"log"
	"net"
	"strings"
	"time"
)

// модель сервера
type server struct {
	rooms    map[string]*room // активные комнаты
	commands chan command     // канал поступающих команд
	online   []*client        // пользователи в сети
}

// newServer возвращает новый сервер
func newServer() *server {
	return &server{
		rooms:    make(map[string]*room),
		commands: make(chan command),
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
	defer listener.Close()
	fmt.Println("Сервер запущен на порте: " + fmt.Sprint(port))
	// для каждого подключающегося клиента создается модель newClient
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println(err)
			return
		}
		go s.newClient(conn, db)
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
		case CMD_QUIT:
			s.quit(cmd.client)
		}
	}
}

// newClient функция возвращает модель клиента
func (s *server) newClient(conn net.Conn, db *mongo.Client) {
	log.Printf("Новый пользователь: %s", conn.RemoteAddr().String())

	c := &client{
		conn:     conn,
		commands: s.commands,
	}

	c.readInput(db)
}

// login функция отвечает за авторизацию пользователя
func (s *server) login(c *client, args []string, db *mongo.Client) {
	//контекст -  если операция не выполнится за 80 секунд - отмена
	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Second)
	defer cancel()
	fmt.Println("Новая попытка входа: ", args)
	fmt.Println(s.online)

	// проверка авторизована ли эта учетная запись, и отключение авторизованног клиента в этом случае
	for i, v := range s.online {
		fmt.Println(args[1], v)
		if args[1] == v.username {
			v.msg("Учетная запись исползована на другом пк, отключение")
			v.conn.Close()
			// удаление авторизованного клиента и списка онлайн пользователей
			s.online = append(s.online[:i], s.online[i:]...)
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
		s.online = append(s.online, c)
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
	users_db := db.Database("users")
	user_collection := users_db.Collection("users")

	hasUsername := user_collection.FindOne(ctx, bson.M{ // проверка существует ли пользователь
		"username": args[1],
	})

	if hasUsername.Err() == nil { // если имя занято
		c.err(errors.New("имя пользователя уже использовано"))
		return
	}

	_, err := user_collection.InsertOne(ctx, bson.D{ // добавлени пользователя в бд
		{Key: "username", Value: args[1]},
		{Key: "password", Value: args[2]},
	})
	if err != nil {
		c.err(errors.New("ошибка базы данных"))
		return
	}
	c.msg(fmt.Sprintf("успешная регистрация"))
	c.username = args[1]           // установка имени пользователя
	s.online = append(s.online, c) // добавление в онлайн список
	c.loggedIn = true
}

// join отвечает за вход в комнату
func (s *server) join(c *client, args []string) {
	roomName := args[1]
	r, ok := s.rooms[roomName] // если комната не существует создаем и добавляем в списсок
	if !ok {
		r = &room{
			name:    roomName,
			members: make(map[net.Addr]*client),
		}
		s.rooms[roomName] = r
	}

	r.members[c.conn.RemoteAddr()] = c // добавление пользователя в список активных в комнате
	s.quitCurrentRoom(c)               // отключение от нынешней комнаты
	c.room = r                         // установка комнаты пользователя
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
	log.Printf("!!! клиент отключился: %s", c.conn.RemoteAddr().String())

	s.quitCurrentRoom(c)         // выход из комнаты
	for i, v := range s.online { // удаление из списка онлайн пользователей
		if v == c {
			s.online = append(s.online[:i], s.online[i+1:]...)
		}
	}
	c.msg("!!! отключение от сервера")
	c.conn.Close() // закрытие соединения
}

// quitCurrentRoom отвечает за выход из комнаты
func (s *server) quitCurrentRoom(c *client) {
	if c.room != nil { // если пользователь в комнате
		delete(c.room.members, c.conn.RemoteAddr()) // удаляем пользователя из списка активных в комнате
		fmt.Println(c.room.members, len(c.room.members))
		if len(c.room.members) == 0 { // если комната пустая - она удаляется
			fmt.Println(s.rooms)
			delete(s.rooms, c.room.name) // удаление комнаты
			fmt.Println(s.rooms)
		} else {
			c.room.broadcast(fmt.Sprintf("%s вышел из комнаты", c.username))
		}
	}
}
