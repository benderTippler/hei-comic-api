package redis

import (
	"fmt"
	"github.com/go-redis/redis/v9"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
)

var (
	redisDb *redis.Client
)

func InitRedis() error {
	ctx := gctx.New()
	redisVar, err := g.Cfg().Get(ctx, "redis")
	if err != nil {
		return err
	}
	redisCfg := redisVar.MapStrVar()

	redisDb = redis.NewClient(&redis.Options{
		Addr:     redisCfg["addr"].String(),
		Password: redisCfg["password"].String(),
		DB:       redisCfg["db"].Int(),
	})
	_, err = redisDb.Ping(ctx).Result()
	fmt.Println("InitRedis")
	return err
}

func NewRedis() *redis.Client {
	return redisDb
}
