package tcpServer

import (
	"context"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
)

// модель пользователя для авторизации
type User struct {
	Username string `bson:"username,omitempty"` // имя
	Password string `bson:"password,omitempty"` // пароль
}

// getdb отвечает за подлючение к бд
func getdb(mongodb string) *mongo.Client {
	client, err := mongo.NewClient(options.Client().ApplyURI(mongodb)) // подкючение к mongoDB
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.TODO() // возвращает пустой контекст для  для комнад
	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}

	return client
}
