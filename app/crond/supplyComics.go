package crond

import (
	"context"
	"errors"
	"fmt"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gfile"
	"github.com/gogf/gf/v2/text/gstr"
	"go.mongodb.org/mongo-driver/bson"
	"hei-comic-api/app/collector/adapter"
	"hei-comic-api/app/model"
	"hei-comic-api/app/model/mongo"
	"hei-comic-api/app/utills"
	"hei-comic-api/base/mysql"
	"math/rand"
	"sync"
)

/**
补充漫画资源脚本，把损坏服务器资源的数据，当道最好，慢慢补齐
*/

var SupplyComicsCrond = new(supplyComics)

type supplyComics struct{}

/*
SupplyChapters
查询漫画状态为5的数据，去补充资源信息，补充完毕后，更改状态为上架状态
*/
func (c *supplyComics) SupplyChapters() error {
	OnShelfComicCrond.updateEnComic(7, 2)
	filter := make(bson.M)
	filter["status"] = 5
	comics, err := Common.getComics(filter)
	if err != nil {
		return err
	}
	if len(comics) == 0 {
		return errors.New("已经没有输出可用采集了")
	}
	db := mysql.NewDb()
	//开始 补齐章节对应的资产数据
	wg := sync.WaitGroup{}
	taskChan := make(chan bool, 1)
	for _, comic := range comics {
		wg.Add(1)
		taskChan <- true
		go func(comic *mongo.Comic) {
			defer wg.Done()
			orderId := comic.OrderId
			uuid := comic.UUID
			TotalChapters := make([]*model.Chapter, 0)
			err = db.Table((&model.Chapter{}).GetTableName(orderId)).Where("pid = ?", uuid).Find(&TotalChapters).Error
			if err != nil {
				<-taskChan
				return
			}
			// 开启协程采集章节
			wgChapter := sync.WaitGroup{}
			taskChapterChan := make(chan bool, 5)
			for _, chapter := range TotalChapters {
				wgChapter.Add(1)
				taskChapterChan <- true
				go func(chapter *model.Chapter) {
					defer wgChapter.Done()
					if chapter.State == 0 || chapter.Resources == "" || chapter.State == 2 {
						if chapter.Resources != "" {
							db.Table((&model.Chapter{}).GetTableName(orderId)).Where("uuid = ?", chapter.UUID).Updates(map[string]interface{}{
								"state": 1,
							})
							<-taskChapterChan
							return
						}

						var resources = make([]string, 0)
						if chapter.Resources != "" {
							resources = gstr.Split(chapter.Resources, "|")
						} else {
							resources, _ = adapter.NewAdapterCollector(orderId).GetResource(chapter.Target)
						}
						if len(resources) > 0 {
							num := rand.Intn(len(resources))
							imgUrl := resources[num]
							if !utills.CheckResource(imgUrl, chapter.Origin) {
								chapter.State = 2
							} else {
								chapter.State = 1
							}
							chapter.Resources = gstr.Join(resources, "|")
						} else {
							chapter.State = 2
						}
						db.Table((&model.Chapter{}).GetTableName(orderId)).Where("uuid = ?", chapter.UUID).Updates(map[string]interface{}{
							"state":     chapter.State,
							"resources": chapter.Resources,
						})
					}
					<-taskChapterChan
				}(chapter)
			}
			wgChapter.Wait()
			//资源过滤完毕，修正漫画状态为2
			Common.updateComic(comic.UUID, comic.OrderId, 2)
			<-taskChan
		}(comic)
	}
	wg.Wait()
	//结束 补齐章节对应的资产数据
	return nil
}

/*
SealedComics
封版漫画。 定时扫描完结漫画资源是否完整并且是否已经保存到本地。然后全部通过，直接封版资源。
*/
func (c *supplyComics) SealedComics() error {
	ctx := context.TODO()
	comicVar, err := g.Cfg().Get(ctx, "comic")
	if err != nil {
		return err
	}
	comicCfg := comicVar.MapStrVar()
	storageDirectory := comicCfg["storageDirectory"].String()

	filter := make(bson.M)
	filter["status"] = 2 //资源可用
	//filter["state"] = 2  //完结
	filter["isHandle"] = 0
	filter["orderId"] = bson.M{
		"$in": []int{
			2, 6, 8, 9,
		},
	}
	comics, err := Common.getComics(filter)
	if err != nil {
		return err
	}
	db := mysql.NewDb()
	//开始 检查数据是否可用封版
	wg := sync.WaitGroup{}
	taskChan := make(chan bool, 5)
	for _, comic := range comics {
		wg.Add(1)
		taskChan <- true
		go func(comic *mongo.Comic) {
			defer wg.Done()
			orderId := comic.OrderId
			uuid := comic.UUID
			// 单独下载一下文件
			adapter.NewAdapterCollector(orderId).ComicToNas(comic)
			//是否存在有 state 为 0,1,2的情况
			var otherCount int64
			err = db.Table((&model.Chapter{}).GetTableName(orderId)).Where("pid = ? and state in ?", uuid, []int{0, 1}).Count(&otherCount).Error
			if err != nil || otherCount > 0 {
				<-taskChan
				return
			}

			TotalChapters := make([]*model.Chapter, 0)
			err = db.Table((&model.Chapter{}).GetTableName(orderId)).Where("pid = ?", uuid).Find(&TotalChapters).Error
			if err != nil {
				<-taskChan
				return
			}
			if len(TotalChapters) == 0 {
				<-taskChan
				return
			}

			// 开启协程采集章节 对比
			doneChapters := make([]*model.Chapter, 0)
			wgChapter := sync.WaitGroup{}
			taskChapterChan := make(chan bool, 5)
			for _, chapter := range TotalChapters {
				wgChapter.Add(1)
				taskChapterChan <- true
				go func(chapter *model.Chapter) {
					defer wgChapter.Done()
					if chapter.State == 3 || chapter.State == 2 {
						paths := gstr.Split(chapter.DownloadPath, "|")
						isUsed := true //资源可用
						for _, path := range paths {
							absolutePath := fmt.Sprintf("%v/%v", storageDirectory, path)
							size := gfile.Size(absolutePath)
							if !gfile.Exists(absolutePath) || size <= 900 { //文件不正常
								isUsed = false
								break
							}
						}
						if isUsed {
							doneChapters = append(doneChapters, chapter)
							if chapter.State == 2 {
								db.Table((&model.Chapter{}).GetTableName(orderId)).Where("uuid = ?", chapter.UUID).Update("state", 3)
							}
						} else { //章节资源有问题
							fmt.Println(chapter.Name, ",", chapter.UUID, "章节图片有问题")
							db.Table((&model.Chapter{}).GetTableName(orderId)).Where("uuid = ?", chapter.UUID).Update("state", 2)
						}
					}
					<-taskChapterChan
				}(chapter)
			}
			wgChapter.Wait()
			// 结束协程采集章节 对比
			//下载到本地数据和总章节数据相等
			if len(doneChapters) == len(TotalChapters) {
				if comic.State == 2 { //完结版漫画才会封版
					Common.updateComic(comic.UUID, comic.OrderId, 88) //正式上线，漫画
				}
			} else { //章节有损坏
				Common.updateIsHandleComic(comic.UUID, comic.OrderId, 44)
			}
			// 执行完毕，释放通道
			<-taskChan
		}(comic)
	}
	wg.Wait()
	//结束 检查数据是否可用封版
	fmt.Println("脚本--> 封版漫画--> 结束")
	return nil
}

/*
FiltrateComics
过滤线上资源损坏的章节漫画，需要后台显示，手动补全
*/
func (c *supplyComics) FiltrateComics() error {
	filter := make(bson.M)
	filter["status"] = 2
	comics, err := Common.getComics(filter)
	if err != nil {
		return err
	}
	if len(comics) == 0 {
		return errors.New("已经没有输出可用采集了")
	}
	db := mysql.NewDb()
	//开始 补齐章节对应的资产数据
	wg := sync.WaitGroup{}
	taskChan := make(chan bool, 5)
	for _, comic := range comics {
		wg.Add(1)
		taskChan <- true
		go func(comic *mongo.Comic) {
			defer wg.Done()
			orderId := comic.OrderId
			uuid := comic.UUID
			var TotalChapters int64
			err = db.Table((&model.Chapter{}).GetTableName(orderId)).Where("pid = ? and state = 2", uuid).Count(&TotalChapters).Error
			if err != nil {
				<-taskChan
				return
			}
			if TotalChapters > 0 {
				Common.updateIsHandleComic(uuid, orderId, 44)
			}
			<-taskChan
		}(comic)
	}
	wg.Wait()
	//结束 补齐章节对应的资产数据
	return nil
}
