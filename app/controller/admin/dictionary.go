package admin

import (
	"context"
	"github.com/labstack/echo"
	"hei-comic-api/app/httpio/admin/in"
	"hei-comic-api/app/middleware"
	"hei-comic-api/app/repo"
)

type dictionary struct {
	Base
}

func NewDictionary() *dictionary {
	c := new(dictionary)
	c.Ctx = middleware.MyCtx{
		Context: context.TODO(),
	}
	return c
}

// CreateDict  创建字典内容
func (c *dictionary) CreateDict(ctx echo.Context) error {
	c.Ctx.EchoCtx = ctx
	req := new(in.CreateDictIn)
	err := ctx.Bind(req)
	if err != nil {
		return err
	}
	err = repo.DictionaryRepo.CreateDict(c.Ctx, req)
	if err != nil {
		return err
	}
	return c.webSuccess(ctx, nil)
}

// FindDict  查询所有字典数据
func (c *dictionary) FindDict(ctx echo.Context) error {
	c.Ctx.EchoCtx = ctx
	rsp, err := repo.DictionaryRepo.FindDict(c.Ctx)
	if err != nil {
		return err
	}
	return c.webSuccess(ctx, rsp)
}
