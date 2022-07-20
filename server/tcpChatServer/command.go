package tcpChatServer

import "go.mongodb.org/mongo-driver/mongo"

type commandID int

// id существующих команд
const (
	CMD_LOGIN commandID = iota // инкремент
	CMD_SIGNUP
	CMD_JOIN
	CMD_ROOMS
	CMD_MSG
	CMD_QUIT
	CMD_DOWNLOAD
)

// модель команды
type command struct {
	id     commandID     // id комманды
	client *client       // указатель на модель пользователя
	args   []string      // аргументы комнады
	db     *mongo.Client // соединение с бд
}
