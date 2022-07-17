package tcpServer

import (
	"bufio"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"net"
	"strings"
)

// Модель клинта
type client struct {
	conn     net.Conn       // TCP соединение
	username string         // имя пользователя
	room     *room          // в какой комнате
	commands chan<- command // команды посланные пользователем
	loggedIn bool           // флаг авторизации
}

// readInput отвечает за чтение комманд пользователя
func (c *client) readInput(db *mongo.Client) {
	for {

		msg, err := bufio.NewReader(c.conn).ReadString('\n') // чтение из соединения
		if err != nil {
			return
		}

		msg = strings.Trim(msg, "\r\n")

		args := strings.Split(msg, " ")
		cmd := strings.TrimSpace(args[0])

		switch cmd { // выполнение блока в зависимости от команды
		case "/login": // авторизация
			if c.loggedIn { // если уже авторизован
				c.err(fmt.Errorf("уже авторизован: %s", cmd))
				break
			}
			c.commands <- command{ // отправление команды в канал команд пользователя для выполнения
				id:     CMD_LOGIN,
				client: c,
				args:   args,
				db:     db,
			}
		case "/signUP": //  регистрация
			if c.loggedIn {
				c.err(fmt.Errorf("уже авторизован: %s", cmd))
				break
			}
			c.commands <- command{
				id:     CMD_SIGNUP,
				client: c,
				args:   args,
				db:     db,
			}
		case "/join": // вход в комнату
			if !c.loggedIn { // если не авторизован
				c.err(fmt.Errorf("не авторизован: %s", cmd))
				break
			}
			c.commands <- command{
				id:     CMD_JOIN,
				client: c,
				args:   args,
			}
		case "/rooms": // список комнат
			if !c.loggedIn {
				c.err(fmt.Errorf("не авторизован: %s", cmd))
				break
			}
			c.commands <- command{
				id:     CMD_ROOMS,
				client: c,
				args:   args,
			}
		case "/msg": // отправка сообщения
			if !c.loggedIn {
				c.err(fmt.Errorf("не авторизован: %s", cmd))
				break
			}
			c.commands <- command{
				id:     CMD_MSG,
				client: c,
				args:   args,
			}
		case "/quit": // отключение
			if !c.loggedIn {
				c.err(fmt.Errorf("не авторизован: %s", cmd))
				break
			}
			c.commands <- command{
				id:     CMD_QUIT,
				client: c,
				args:   args,
			}
		default: // если команда не существует
			c.err(fmt.Errorf("такой команды нет: %s", cmd))
		}
	}
}

// отправка ошибки пользоватлю
func (c *client) err(err error) {
	c.conn.Write([]byte("ERR: " + err.Error() + "\n"))
}

// отправка сообщения пользователю
func (c *client) msg(msg string) {
	c.conn.Write([]byte(msg + "\n"))
}
