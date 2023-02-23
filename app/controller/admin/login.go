package admin

import (
	"context"
	"github.com/gogf/gf/v2/encoding/gbase64"
	"github.com/gogf/gf/v2/util/grand"
	"github.com/gogf/gf/v2/util/guid"
	"github.com/labstack/echo"
	"hei-comic-api/app/httpio/admin/out"
	"hei-comic-api/app/middleware"
	"hei-comic-api/base/cache"
	"time"
)

type loginBackend struct {
	Base
}

func NewLoginBackend() *loginBackend {
	u := new(loginBackend)
	u.Ctx = middleware.MyCtx{
		Context: context.TODO(),
	}
	return u
}

// Captcha 图形验证码
func (login *loginBackend) Captcha(ctx echo.Context) error {
	login.Ctx.EchoCtx = ctx
	capchaOut := &out.CapchaOut{}
	captchaText := grand.Digits(4)
	svg := `<svg width="150" height="50" xmlns="http://www.w3.org/2000/svg"><text x="75" y="25" text-anchor="middle" font-size="25" fill="#fff">` + captchaText + `</text></svg>`
	svgbase64 := gbase64.EncodeString(svg)
	capchaOut.Data = `data:image/svg+xml;base64,` + svgbase64
	capchaOut.CaptchaId = guid.S()
	cache.CacheManager.Set(login.Ctx.Context, capchaOut.CaptchaId, captchaText, 1800*time.Second)
	return login.webSuccess(ctx, capchaOut)
}
