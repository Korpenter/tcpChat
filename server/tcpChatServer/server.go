package tcpChatServer

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

const FileStorageRoot = "server_storage"

var dbURI string // флаг порта, если не введен, 8888

var F *os.File

// модель сервера
type server struct {
	rooms    map[string]*room     // активные комнаты
	commands chan command         // канал поступающих команд
	online   map[net.Conn]*client // пользователи в сети
	tempmap  map[net.Conn]*client // пользователи до авторизации
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

// Start запускает вызывает рутину запуска сервера
func Start(port int) {
	go func() { serverStart(port) }()
}

// serverStart запускает сервер на заданном порте
func serverStart(port int) {
	flag.StringVar(&dbURI, "db", "mongodb+srv://Doronin4941:PracticePass@cluster0.05xmh.mongodb.net/?retryWrites=true&w=majority", "db URI")
	log.SetOutput(F)
	s := newServer()
	go s.run() // запуск рутин обработки команд
	log.Println("running on port: " + fmt.Sprint(port))
	// подключение к бд
	db := getDB(dbURI)
	listener, err := net.Listen("tcp", ":"+fmt.Sprint(port)) // запуск на TCP socket для приема входящих соединений
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Println("Сервер запущен на порте: " + fmt.Sprint(port))
	defer listener.Close()
	for {
		conn, err := listener.Accept() // для каждого поступаещго соединения
		if err != nil {
			log.Fatal(err)
		}
		log.Println("new connection found!", conn.RemoteAddr().String())
		s.regConn(conn, db) // вызывается функция регистрации
	}
}

// regConn регистрирует новые соединения
func (s *server) regConn(conn net.Conn, db *mongo.Client) {
	log.SetOutput(F)
	conn.Write([]byte("50" + conn.RemoteAddr().String() + "\n"))
	for {
		msg, err := bufio.NewReader(conn).ReadString('\n') // чтение из соединения
		if err != nil {
			log.Println(err)
			return
		}
		msg = strings.Trim(msg, "\r\n")
		args := strings.Split(msg, " ")
		code := strings.TrimSpace(args[0])
		switch code {
		case "100":
			s.newClient(conn, db) // регистрация первого соединения клиента
			return
		case "105":
			for _, c := range s.tempmap {
				if args[1] == c.chatConn.RemoteAddr().String() {
					c.fileConn = conn // регистрация второго соединения клиента
					log.Println("second client connection registered:", c.fileConn.RemoteAddr().String())
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
		case cmdLogin:
			s.login(cmd.client, cmd.args, cmd.db)
		case cmdSignup:
			s.signUP(cmd.client, cmd.args, cmd.db)
		case cmdJoin:
			s.join(cmd.client, cmd.args)
		case cmdRooms:
			s.listRooms(cmd.client)
		case cmdMsg:
			s.msg(cmd.client, cmd.args)
		case cmdDownload:
			s.sendFileMsg(cmd.client, cmd.args)
		case cmdStartSend:
			s.sendFileData(cmd.client, cmd.args)
		case cmdStartsGet:
			s.getFile(cmd.client, cmd.args)
		case cmdFiles:
			s.listFiles(cmd.client)
		case cmdQuit:
			s.quit(cmd.client)
		}
	}
}

// newClient функция возвращает модель клиента
func (s *server) newClient(conn net.Conn, db *mongo.Client) {
	log.SetOutput(F)
	log.Printf("new user: %s", conn.RemoteAddr().String())

	c := &client{
		chatConn: conn,
		commands: s.commands,
	}
	s.tempmap[c.chatConn] = c
	go c.readInput(db)
}

// login функция отвечает за авторизацию пользователя
func (s *server) login(c *client, args []string, db *mongo.Client) {
	log.SetOutput(F)
	//контекст -  если операция не выполнится за 80 секунд - отмена
	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Second)
	defer cancel()
	log.Println("new login: ", args)
	// проверка авторизована ли эта учетная запись, и отключение авторизованног клиента в этом случае
	for _, v := range s.online {
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
		c.notif("Введите логин и пароль")
		return
	}

	user := userCollection.FindOne(ctx, bson.M{"username": args[1]}) // поиск по БД по имени пользователя
	var userField bson.M
	err := user.Err()
	if err != nil {
		log.Println(err.Error())
		c.notif("Пользователь не найден")
		return
	}

	err = user.Decode(&userField) // раскодирование из BSON
	if err != nil {
		c.notif("Ошибка базы данных" + err.Error())
		return
	}

	if userField["password"] == args[2] { // проверка пароля
		c.msg("201")
		c.loggedIn = true
		c.username = args[1]
		delete(s.tempmap, c.chatConn)
		s.online[c.chatConn] = c
		return
	} else {
		c.notif("Неверный пароль")
		return
	}
}

// signUP отвечает за авториза
func (s *server) signUP(c *client, args []string, db *mongo.Client) {
	log.SetOutput(F)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if len(args) != 3 {
		c.notif("Введите логин и пароль")
		return
	}
	log.Println("new signup: ", args)
	usersDb := db.Database("users")
	userCollection := usersDb.Collection("users")

	hasUsername := userCollection.FindOne(ctx, bson.M{ // проверка существует ли пользователь
		"username": args[1],
	})

	if hasUsername.Err() == nil { // если имя занято
		c.notif("Имя пользователя уже использовано")
		return
	}

	_, err := userCollection.InsertOne(ctx, bson.D{ // добавлени пользователя в бд
		{Key: "username", Value: args[1]},
		{Key: "password", Value: args[2]},
	})
	if err != nil {
		c.notif("Ошибка базы данных")
		return
	}
	c.msg("202")
	c.username = args[1]     // установка имени пользователя
	s.online[c.chatConn] = c // добавление в онлайн список
	c.loggedIn = true
}

// join отвечает за вход в комнату
func (s *server) join(c *client, args []string) {
	log.SetOutput(F)
	if len(args) != 2 {
		c.notif("неверный формат команды")
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
	r.broadcast(fmt.Sprintf(" > %s Присоединился", c.username))
	c.notif(fmt.Sprintf("в %s", r.name))
}

// listRooms возвращает список активных комнат
func (s *server) listRooms(c *client) {
	var rooms []string
	for name := range s.rooms {
		rooms = append(rooms, name)
	}

	c.msg(fmt.Sprintf("Доступные комнаты:%s", strings.Join(rooms, ", ")))
}

// msg отправляет сообщение всем пользователя в комнате
func (s *server) msg(c *client, args []string) {
	if c.room == nil {
		c.notif("Не в комнате")
		return
	}

	c.room.broadcast(c.username + ": " + strings.Join(args[1:], " "))
}

// quit отвечает за отключение от сервера и закрытие соединения
func (s *server) quit(c *client) {
	log.SetOutput(F)
	log.Printf("client disconnected: %s", c.username)

	s.quitCurrentRoom(c)         // выход из комнаты
	for _, v := range s.online { // удаление из списка онлайн пользователей
		if v == c {
			delete(s.online, c.chatConn)
		}
	}
	c.notif("Отключение от сервера")
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

//
func (s *server) sendFileMsg(c *client, args []string) {
	log.SetOutput(F)
	if len(args) != 2 {
		c.notifFile(fmt.Sprintf("Неверный формат команды %v", args))
		return
	}
	inputFile, err := os.Open(path.Join(FileStorageRoot, args[1]))
	if err != nil {
		log.Println(err.Error())
		c.notifFile(err.Error())
		return
	}
	defer inputFile.Close()
	stats, _ := inputFile.Stat()
	send := fmt.Sprintf("download %s %d", args[1], stats.Size())
	c.msgFile(send)
	if err != nil {
		log.Println(err)
	}
}

func (s *server) sendFileData(c *client, args []string) {
	log.SetOutput(F)
	if len(args) != 2 {
		c.notifFile(fmt.Sprintf("неверный формат команды %v", args))
		return
	}
	log.Println(fmt.Sprintf("%s sending file: %v", c.username, args[1]))
	inputFile, err := os.Open(path.Join(FileStorageRoot, args[1]))
	if err != nil {
		if os.IsNotExist(err) {
			c.notifFile("Файл не найден")
			return
		}
		log.Println(err.Error())
		c.notifFile(err.Error())
		return
	}

	defer inputFile.Close()

	io.Copy(c.fileConn, inputFile)

	log.Println(fmt.Sprintf("%v sent succesfully by %s", args[1], c.username))
}

// getFile получает файл от клиента и сохраняет на сервере
func (s *server) getFile(c *client, args []string) {
	log.SetOutput(F)
	if len(args) != 3 { // проверка формата команды
		c.notifFile(fmt.Sprintf("| Неверный формат команды %v", args))
		return
	}
	log.Println(fmt.Sprintf("%s start uploading file %v", c.username, args[1]))
	fileSize, err := strconv.ParseInt(args[2], 10, 64) // размер файла
	if err != nil || fileSize == -1 {
		log.Println(err.Error())
		c.notifFile("Ошибка открытия файла или файл не существует")
		return
	}
	fileName := args[1] // имя файла
	// проверка расширения и добавление времени создания для избежания дубликатов
	if strings.IndexByte(fileName, '.') != -1 {
		fileName = fmt.Sprintf("%v_%v%v", fileName[:strings.IndexByte(args[1], '.')], time.Now().UnixMilli(),
			fileName[strings.IndexByte(args[1], '.'):])
	} else {
		fileName = fmt.Sprintf("%v_%v", fileName, time.Now().UnixMilli())
	}

	outputFile, err := os.Create(path.Join(FileStorageRoot, fileName)) // создание файла

	if err != nil {
		log.Println(err.Error())
		c.notifFile(err.Error())
		return
	}
	defer outputFile.Close()
	log.Println("start uploading", args)
	c.msgFile("200 Start upload!") // подтверждения начала загрузки

	io.Copy(outputFile, io.LimitReader(c.fileConn, fileSize)) // запись в файл из соединения

	log.Println(fmt.Sprintf("%v uploaded successfuly by %s", args[1], c.username))
	c.notifFile(fmt.Sprintf("| Успешно загружен на сервер %v", fileName))
}

// listFiles отправляет клиенту список файлов на сервере
func (s *server) listFiles(c *client) {
	log.SetOutput(F)
	var files strings.Builder                        // объект для построения строки
	fileInfo, err := ioutil.ReadDir(FileStorageRoot) // чтение директории
	if err != nil {
		log.Println(err)
		return
	}
	files.WriteString(fmt.Sprintf("Количество файлов - %v\n", len(fileInfo)))
	for i, file := range fileInfo { // построение строки
		files.WriteString(fmt.Sprintf("%v. %v\n", i, file.Name()))
	}
	files.WriteString("|")
	c.msgFile(files.String()) // отправка сообщеения
	log.Println(fmt.Sprintf("file list sent to %s", c.username))
}
