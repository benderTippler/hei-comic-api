package controller

import (
	"context"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/labstack/echo"
	"hei-comic-api/app/httpio/in"
	"hei-comic-api/app/middleware"
	"hei-comic-api/app/repo"
)

type user struct {
	Base
}

func NewUser() *user {
	u := new(user)
	u.Ctx = middleware.MyCtx{
		Context: context.TODO(),
	}
	return u
}

// SendEmailCode 发送邮箱验证码
func (user *user) SendEmailCode(ctx echo.Context) error {
	user.Ctx.EchoCtx = ctx
	req := new(in.SendEmailCode)
	err := ctx.Bind(req)
	if err != nil {
		return err
	}
	err = repo.UserRepo.SendEmailCode(user.Ctx, req)
	if err != nil {
		return err
	}
	return user.webSuccess(ctx, nil)
}

// Register 用户注册
func (user *user) Register(ctx echo.Context) error {
	user.Ctx.EchoCtx = ctx
	req := new(in.Register)
	err := ctx.Bind(req)
	if err != nil {
		return err
	}
	err = ctx.Validate(req)
	if err != nil {
		return err
	}
	err = repo.UserRepo.Register(user.Ctx, req)
	if err != nil {
		return err
	}
	return user.webSuccess(ctx, nil)
}

// Login 用户登录
func (user *user) Login(ctx echo.Context) error {
	user.Ctx.EchoCtx = ctx
	req := new(in.Login)
	err := ctx.Bind(req)
	if err != nil {
		return err
	}
	rep, err := repo.UserRepo.Login(user.Ctx, req)
	if err != nil {
		return err
	}
	return user.webSuccess(ctx, rep)
}

// UserInfo 用户信息
func (user *user) UserInfo(ctx echo.Context) error {
	user.Ctx.EchoCtx = ctx
	uuid := ctx.Get("uuid").(string)
	rep, err := repo.UserRepo.UserInfo(user.Ctx, gconv.Int64(uuid))
	if err != nil {
		return err
	}
	return user.webSuccess(ctx, rep)
}
