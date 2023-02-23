package controller

import (
	"encoding/json"
	"fmt"
	"github.com/gogf/gf/v2/crypto/gmd5"
	"github.com/labstack/echo"
	"hei-comic-api/app/httpio/out"
	"hei-comic-api/app/middleware"
	"hei-comic-api/app/utills/encrypt"
	"net/http"
	"time"
)

type Base struct {
	Ctx middleware.MyCtx
}

// WebSuccess 统一处理结果返回
func (b *Base) webSuccess(ctx echo.Context, data interface{}) error {
	if data == nil {
		return ctx.JSON(http.StatusOK, &out.Result{
			Code:    http.StatusOK,
			Secret:  "",
			IV:      "",
			Data:    "",
			Message: "请求接口成功",
		})
	}
	secret := []byte(gmd5.MustEncryptString(fmt.Sprintf("heibox-%v", time.Now().Nanosecond())))
	rspSrt, err := encrypt.BaseEncrypt.RSAEncrypt(secret)
	if err != nil {
		return err
	}
	src, err := json.Marshal(data)
	if err != nil {
		return err
	}
	result, iv, err := encrypt.BaseEncrypt.AESEncrypt(src, secret)
	if err != nil {
		return err
	}
	rspIV, err := encrypt.BaseEncrypt.RSAEncrypt(iv)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, &out.Result{
		Code:    http.StatusOK,
		Secret:  rspSrt,
		IV:      rspIV,
		Data:    result,
		Message: "请求接口成功",
	})
}
