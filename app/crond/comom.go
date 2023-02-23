package crond

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"hei-comic-api/app/model"
	"hei-comic-api/app/model/mongo"
	baseMongo "hei-comic-api/base/mongo"
	"hei-comic-api/base/mysql"
)

var Common = new(common)

type common struct{}

func (c *common) getComics(filter bson.M) ([]*mongo.Comic, error) {
	ctx := context.TODO()
	mongoClient := baseMongo.NewMongo().Database("comics").Collection("comics")
	comics := make([]*mongo.Comic, 0, 21)
	var findOpts = &options.FindOptions{}
	sorts := bson.D{{"updateTime", -1}}
	findOpts.SetSort(sorts)
	cur, err := mongoClient.Find(ctx, filter, findOpts)
	if err != nil {
		return comics, err
	}
	defer cur.Close(ctx)
	err = cur.All(ctx, &comics)
	if err != nil {
		return comics, err
	}
	return comics, nil
}

// 更新数据库和mongo 数据
func (c *common) updateComic(uuid int64, orderId int, status int) {
	mongoClient := baseMongo.NewMongo().Database("comics").Collection("comics")
	db := mysql.NewDb()
	db.Table((&model.Comic{}).GetTableName(orderId)).Where("uuid = ?", uuid).Update("status", status)
	filer := bson.D{{"uuid", uuid}}
	update := bson.M{
		"$set": bson.M{
			"status": status,
		},
	}
	upsert := false
	mongoClient.UpdateOne(context.TODO(), filer, update, &options.UpdateOptions{
		Upsert: &upsert,
	})
}

func (c *common) updateIsHandleComic(uuid int64, orderId int, isHandle int) {
	mongoClient := baseMongo.NewMongo().Database("comics").Collection("comics")
	db := mysql.NewDb()
	db.Table((&model.Comic{}).GetTableName(orderId)).Where("uuid = ?", uuid).Update("isHandle", isHandle)
	filer := bson.D{{"uuid", uuid}}
	update := bson.M{
		"$set": bson.M{
			"isHandle": isHandle,
		},
	}
	upsert := false
	mongoClient.UpdateOne(context.TODO(), filer, update, &options.UpdateOptions{
		Upsert: &upsert,
	})
}
