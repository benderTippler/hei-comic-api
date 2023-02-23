package middleware

import (
	"context"
	"fmt"
	"github.com/gogf/gf/v2/crypto/gmd5"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/labstack/echo"
	baseError "hei-comic-api/app/error"
	"hei-comic-api/base/redis"
)

func CheckUser(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		//在这里处理拦截请求的逻辑
		token := c.Request().Header.Get("Authorization")
		token = gstr.ReplaceByMap(token, map[string]string{
			"Bearer ": "",
		})
		uuid := c.Get("uuid").(string)
		redisToken, _ := redis.NewRedis().Get(context.TODO(), fmt.Sprintf("%v-%v", "login", uuid)).Result()
		if redisToken != gmd5.MustEncryptString(token) { //不是最新用户登录的token，强制用户下线
			fmt.Println(redisToken, gmd5.MustEncryptString(token), token)
			return baseError.JwtInvalidErr
		}
		fmt.Println(uuid, token)
		//执行下一个中间件或者执行控制器函数, 然后返回执行结果
		return next(c)
	}
}
