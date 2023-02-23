package bank

import (
	"gorm.io/gorm"
	"hei-comic-api/app/collector/adapter"
	"hei-comic-api/app/model"
)

var DemoAdapter = new(demo)

type demo struct {
	adapter.BaseAdapter
}

// 初始化包子实例对象
func init() {
	//BaoZiAdapter.Order = 2
	//BaoZiAdapter.Origin = "https://manhua.dmzj.com"
	//BaoZiAdapter.Referer = "https://manhua.dmzj.com"
	//BaoZiAdapter.MaxTry = 5
	//BaoZiAdapter.CachePath = "./dataCache"

}

// StartUp 采集脚本入口
func (a *demo) StartUp(globalDb *gorm.DB) error {
	return nil
}

// 漫画列表页面
func (a *demo) List(page int) ([]*model.Comic, error) {
	var err error
	comics := make([]*model.Comic, 0)

	return comics, err
}

// 漫画详情
func (a *demo) Details(comic *model.Comic) error {
	var err error
	return err
}

// 获取章节总数
func (a *demo) ChapterCount(comic *model.Comic) (int, error) {
	count := 0
	var err error
	return count, err
}

// 获取章节数列表
func (a *demo) Chapters(comic *model.Comic) ([]*model.Chapter, error) {
	var err error
	chapters := make([]*model.Chapter, 0)
	return chapters, err
}

// 漫画资源
func (a *demo) GetResource(chapter *model.Chapter) ([]string, error) {
	var err error
	resource := make([]string, 0)
	return resource, err
}
