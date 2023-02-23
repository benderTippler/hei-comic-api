package bank

import (
	"github.com/gogf/gf/v2/container/gset"
	"hei-comic-api/app/collector/adapter"
	"hei-comic-api/app/model"
)

var EnAdapter = new(enDemon)

type enDemon struct {
	adapter.BaseAdapter
}

// 初始化包子实例对象
func init() {
	err := EnAdapter.InitCfg(99) //初始化配置
	if err != nil {
		panic("BaoZiAdapter InitCfg Fail")
	}
}

// StartUp 采集器启动
func (a *enDemon) StartUp() error {
	return nil
}

func (a *enDemon) GetUpdateComics() ([]*model.Comic, error) {
	comicSet := gset.New(true)
	comics := make([]*model.Comic, 0)
	//targetUrl := fmt.Sprintf("%v/list/new", a.Origin)

	comicSet.Clear()
	return comics, nil
}

// List 漫画列表页面 完成
func (a *enDemon) List(target string) ([]*model.Comic, error) {
	var err error
	comics := make([]*model.Comic, 0)

	return comics, err
}

// Details 漫画详情 完成
func (a *enDemon) Details(comic *model.Comic) error {
	var err error

	return err
}

// ChapterCount 获取章节总数 完成
func (a *enDemon) ChapterCount(comic *model.Comic) (int, error) {
	count := 0
	var err error

	return count, err
}

// Chapters 获取章节数列表
func (a *enDemon) Chapters(comic *model.Comic) ([]*model.Chapter, error) {
	var err error
	chapters := make([]*model.Chapter, 0)

	return chapters, err
}

// GetResource 漫画资源 完成
func (a *enDemon) GetResource(targetUrl string) ([]string, error) {
	var err error
	resource := make([]string, 0)
	return resource, err
}
