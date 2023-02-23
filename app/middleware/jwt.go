package middleware

import (
	"context"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/golang-jwt/jwt/v4"
	"time"
)

type CustomUserJwt struct {
	UUID     string `json:"uuid"`
	UserType string `json:"userType"` //用户类型 admin 后台管理 user 前台用户
	jwt.RegisteredClaims
}

func (c *CustomUserJwt) GetAccessToken(ctx context.Context) (string, error) {
	var tokenStr string
	c.ExpiresAt = jwt.NewNumericDate(time.Now().Add(time.Hour * 72))
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	// 获取 jwt加密秘钥
	jwtVar, err := g.Cfg().Get(ctx, "jwt")
	if err != nil {
		return tokenStr, err
	}
	jwtConf := jwtVar.MapStrVar()
	tokenStr, err = token.SignedString(jwtConf["secret"].Bytes())
	if err != nil {
		return tokenStr, err
	}
	return tokenStr, nil
}
