package tcpChatServer

import (
	"net"
)

// модель комнаты
type room struct {
	name    string               // название комнаты
	members map[net.Addr]*client // пользователи в комнате
}

// отправка сообщения все пользователя в комнате
func (r *room) broadcast(msg string) {
	for _, m := range r.members { // каждому пользователю в списке отправляется сообщение
		m.msg(msg)
	}

}
