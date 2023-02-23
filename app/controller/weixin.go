package controller

import "C"
import (
	"context"
	"github.com/labstack/echo"
	baseError "hei-comic-api/app/error"
	"hei-comic-api/app/httpio/in"
	"hei-comic-api/app/middleware"
	"hei-comic-api/app/repo"
	"hei-comic-api/app/utills"
	"net/http"
)

type weiXin struct {
	Base
}

func NewWeiXin() *weiXin {
	c := new(weiXin)
	c.Ctx = middleware.MyCtx{
		Context: context.TODO(),
	}
	return c
}

// Subscribe 关注微信号逻辑处理
func (c *weiXin) Subscribe(ctx echo.Context) error {
	c.Ctx.EchoCtx = ctx
	req := new(in.Subscribe)
	err := ctx.Bind(req)
	if err != nil {
		return err
	}
	// 校验签名
	if !utills.CheckSign(req.Sign, req.FromUserName) {
		return baseError.SignInvalidErr
	}
	rsp, err := repo.VipCodeRepo.Subscribe(c.Ctx, req)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, rsp)
}

// UnSubscribe 取关微信号逻辑处理
func (c *weiXin) UnSubscribe(ctx echo.Context) error {
	c.Ctx.EchoCtx = ctx
	req := new(in.UnSubscribe)
	err := ctx.Bind(req)
	if err != nil {
		return err
	}
	// 校验签名
	if !utills.CheckSign(req.Sign, req.FromUserName) {
		return baseError.SignInvalidErr
	}
	err = repo.VipCodeRepo.UnSubscribe(c.Ctx, req)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, nil)
}
