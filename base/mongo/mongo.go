package mongo

import (
	"context"
	"fmt"
	"github.com/gogf/gf/v2/frame/g"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

var client *mongo.Client

func InitMongo() error {
	ctx := context.TODO()
	mongoVar, err := g.Cfg().Get(ctx, "mongo")
	if err != nil {
		return err
	}
	mongoCfg := mongoVar.MapStrVar()

	url := fmt.Sprintf("mongodb://%v:%v@%v:%d",
		mongoCfg["user"].String(),
		mongoCfg["pass"].String(),
		mongoCfg["host"].String(),
		mongoCfg["port"].Int(),
	)
	clientOpts := options.Client().ApplyURI(url)
	clientOpts.SetMaxConnIdleTime(10 * time.Minute)
	clientOpts.SetMaxPoolSize(100)
	clientOpts.SetMinPoolSize(50)

	// 连接到MongoDB
	client, err = mongo.Connect(ctx, clientOpts)
	if err != nil {
		return err
	}

	// 检查连接
	err = client.Ping(context.TODO(), nil)
	if err != nil {
		return err
	}
	fmt.Println("Connected to MongoDB!")
	return nil
}

func NewMongo() *mongo.Client {
	return client
}
