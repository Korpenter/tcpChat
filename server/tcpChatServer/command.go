package tcpChatServer

import "go.mongodb.org/mongo-driver/mongo"

type commandID int

// id существующих команд
const (
	cmdLogin commandID = iota // инкремент
	cmdSignup
	cmdJoin
	cmdRooms
	cmdMsg
	cmdQuit
	cmdDownload
	cmdStartSend
	cmdStartsGet
	cmdFiles
)

// модель команды
type command struct {
	id     commandID     // id комманды
	client *client       // указатель на модель пользователя
	args   []string      // аргументы комнады
	db     *mongo.Client // соединение с бд
}
