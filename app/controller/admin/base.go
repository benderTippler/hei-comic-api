package admin

import (
	"github.com/labstack/echo"
	"hei-comic-api/app/httpio/out"
	"hei-comic-api/app/middleware"
	"net/http"
)

type Base struct {
	Ctx middleware.MyCtx
}

// WebSuccess 统一处理结果返回
func (b *Base) webSuccess(ctx echo.Context, data interface{}) error {
	return ctx.JSON(http.StatusOK, &out.Result{
		Code:    http.StatusOK,
		Data:    data,
		Message: "请求接口成功",
	})
}
