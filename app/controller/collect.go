package controller

import (
	"context"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/labstack/echo"
	"hei-comic-api/app/httpio/in"
	"hei-comic-api/app/middleware"
	"hei-comic-api/app/repo"
)

type collect struct {
	Base
}

func NewCollect() *collect {
	c := new(collect)
	c.Ctx = middleware.MyCtx{
		Context: context.TODO(),
	}
	return c
}

// CreateCollect 创建收藏
func (c *collect) CreateCollect(ctx echo.Context) error {
	c.Ctx.EchoCtx = ctx
	req := new(in.CollectIn)
	err := ctx.Bind(req)
	if err != nil {
		return err
	}
	uuid := ctx.Get("uuid").(string)
	err = repo.CollectRepo.CreateCollect(c.Ctx, gconv.Int64(uuid), req)
	if err != nil {
		return err
	}
	return c.webSuccess(ctx, nil)
}

// UserCollects 收藏列表
func (c *collect) UserCollects(ctx echo.Context) error {
	c.Ctx.EchoCtx = ctx
	uuid := ctx.Get("uuid").(string)
	rsp, err := repo.CollectRepo.UserCollects(c.Ctx, gconv.Int64(uuid))
	if err != nil {
		return err
	}
	return c.webSuccess(ctx, rsp)
}

// UserIsCollectComicId 漫画是否收藏过
func (c *collect) UserIsCollectComicId(ctx echo.Context) error {
	c.Ctx.EchoCtx = ctx
	req := new(in.UserIsCollectComicIdIn)
	err := ctx.Bind(req)
	if err != nil {
		return err
	}
	uuid := ctx.Get("uuid").(string)
	rsp, err := repo.CollectRepo.UserIsCollectComicId(c.Ctx, gconv.Int64(uuid), req)
	if err != nil {
		return err
	}
	return c.webSuccess(ctx, rsp)
}
