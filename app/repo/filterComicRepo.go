package repo

import (
	"context"
	"errors"
	"fmt"
	"github.com/gogf/gf/v2/frame/g"
	"gorm.io/gorm"
	"hei-comic-api/app/collector/adapter"
	"hei-comic-api/app/model"
	"hei-comic-api/base/mysql"
	"sort"
	"sync"
)

/**
主要是过滤重复漫画资源，减少采集和数据库的压力
1、以漫画之家资源为基础进行去重
2、第二步以mangabz资源过滤包子漫画数据

*/

var (
	FilterComicRepo = new(filterComicRepo)
)

type filterComicRepo struct{}

// FilterComicByName 通过漫画名称过滤,通过mongo聚合数据对比
// true 存在 false 不存在

// FilterComicByName 通过漫画名称过滤
// true 存在 false 不存在
func (r *filterComicRepo) FilterComicByName(name string, orderId int) bool {
	db := mysql.NewDb()
	comicM := &model.Comic{}
	err := db.Table((&model.Comic{}).GetTableName(orderId)).Where("name = ? and status = 0", name).Find(&comicM).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false
		}
		return true
	}
	if comicM.UUID == 0 {
		return false
	}

	var count int64
	db.Table((&model.Chapter{}).GetTableName(orderId)).Where("pid = ?", comicM.UUID).Count(&count)
	if count == 0 {
		return false
	}
	return true
}

func (r *filterComicRepo) Tb() {
	adapter.BaoZiAdapter.TongTable()
	adapter.ManHuaAdapter.TongTable()
	adapter.DuShiAdapter.TongTable()
	adapter.LaiManHuaAdapter.TongTable()
	adapter.MangaBzAdapter.TongTable()
	adapter.MangasTreamAdapter.TongTable()
}

// 数据校验 初始化一次
func (r *filterComicRepo) CheckComics() {
	//修正脚本
	db := mysql.NewDb()
	adapterVar, err := g.Cfg().Get(context.TODO(), "adapter")
	if err != nil {
		return
	}
	adaptersCfg := adapterVar.Vars()
	sort.Slice(adaptersCfg, func(i, j int) bool {
		return adaptersCfg[i].MapStrVar()["sort"].Int() < adaptersCfg[j].MapStrVar()["sort"].Int()
	})
	// 数据资源优先原则，顺序采集
	for _, adapterCfg := range adaptersCfg {
		adapterMap := adapterCfg.MapStrVar()
		isSwitch := adapterMap["switch"].Bool()
		orderId := adapterMap["orderId"].Int()
		fmt.Println(adapterMap["name"], "开始 执行脚本 列表采集")
		if isSwitch {
			adapter := adapter.NewAdapterCollector(adapterMap["orderId"].Int())
			adapter.CollectChapter()
			comics := make([]*model.Comic, 0)
			err = db.Table((&model.Comic{}).GetTableName(orderId)).Where("status = 0").Find(&comics).Error
			if err != nil {
				continue
			}
			wg := sync.WaitGroup{}
			chanTask := make(chan bool, 100)
			for _, comic := range comics {
				wg.Add(1)
				chanTask <- true
				go func(comic *model.Comic) {
					defer wg.Done()
					adapter.UpdateMongo(comic.UUID)
					<-chanTask
				}(comic)
			}
			wg.Wait()
		}
	}

}
