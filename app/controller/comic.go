package controller

import (
	"context"
	_ "github.com/gogf/gf/v2"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/labstack/echo"
	"hei-comic-api/app/httpio/in"
	"hei-comic-api/app/middleware"
	"hei-comic-api/app/repo"
)

type comic struct {
	Base
}

func NewComic() *comic {
	c := new(comic)
	c.Ctx = middleware.MyCtx{
		Context: context.TODO(),
	}
	return c
}

// List 查询漫画列表
func (comic *comic) List(ctx echo.Context) error {
	comic.Ctx.EchoCtx = ctx
	req := new(in.ComicList)
	err := ctx.Bind(req)
	if err != nil {
		return err
	}
	list, err := repo.ComicRepo.FindComicList(comic.Ctx, req)
	if err != nil {
		return err
	}
	return comic.webSuccess(ctx, list)
}

// GetComicById 获取漫画详情
func (comic *comic) GetComicById(ctx echo.Context) error {
	comic.Ctx.EchoCtx = ctx
	uuid := ctx.QueryParam("uuid")
	rsp, err := repo.ComicRepo.GetComicById(comic.Ctx, gconv.Uint64(uuid))
	if err != nil {
		return err
	}
	return comic.webSuccess(ctx, rsp)
}

// GetComicSetting 全局配置
func (comic *comic) GetComicSetting(ctx echo.Context) error {
	comic.Ctx.EchoCtx = ctx
	rsp, err := repo.ComicRepo.GetComicSetting(comic.Ctx)
	if err != nil {
		return err
	}
	return comic.webSuccess(ctx, rsp)
}

// GetComicChapters 漫画章节列表
func (comic *comic) GetComicChapters(ctx echo.Context) error {
	comic.Ctx.EchoCtx = ctx
	req := new(in.GetComicChapters)
	err := ctx.Bind(req)
	if err != nil {
		return err
	}
	rsp, err := repo.ChapterRepo.GetComicChapters(comic.Ctx, req)
	if err != nil {
		return err
	}
	return comic.webSuccess(ctx, rsp)
}

// GetComicChapterResource 获取漫画章节数据
func (comic *comic) GetComicChapterResource(ctx echo.Context) error {
	comic.Ctx.EchoCtx = ctx
	req := new(in.ChapterResource)
	err := ctx.Bind(req)
	rsp, err := repo.ChapterRepo.GetComicChapterResource(comic.Ctx, req)
	if err != nil {
		return err
	}
	return comic.webSuccess(ctx, rsp)
}

// GetComicChapterTop 获取漫画前10章漫画数据
func (comic *comic) GetComicChapterTop(ctx echo.Context) error {
	comic.Ctx.EchoCtx = ctx
	req := new(in.GetComicChapterTops)
	err := ctx.Bind(req)
	rsp, err := repo.ChapterRepo.GetComicChapterTops(comic.Ctx, req)
	if err != nil {
		return err
	}
	return comic.webSuccess(ctx, rsp)
}

// ComparisonComic 后台页面对比数据
func (comic *comic) ComparisonComic(ctx echo.Context) error {
	comic.Ctx.EchoCtx = ctx
	req := new(in.ComparisonComic)
	err := ctx.Bind(req)
	if err != nil {
		return err
	}
	list, err := repo.ComicRepo.ComparisonComic(comic.Ctx, req)
	if err != nil {
		return err
	}
	return ctx.JSON(200, list)
}

// FindDict  查询所有字典数据
func (comic *comic) FindDict(ctx echo.Context) error {
	comic.Ctx.EchoCtx = ctx
	rsp, err := repo.DictionaryRepo.FindDict(comic.Ctx)
	if err != nil {
		return err
	}
	return comic.webSuccess(ctx, rsp)
}

// ComicUpdate 漫画数据
func (comic *comic) ComicUpdate(ctx echo.Context) error {
	comic.Ctx.EchoCtx = ctx
	rep, err := repo.ComicRepo.ComicUpdate(comic.Ctx)
	if err != nil {
		return err
	}
	return comic.webSuccess(ctx, rep)
}
