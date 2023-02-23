package adapter

import (
	"context"
	"errors"
	"fmt"
	"github.com/gogf/gf/v2/container/gset"
	"github.com/gogf/gf/v2/crypto/gmd5"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/gfile"
	"github.com/gogf/gf/v2/text/gstr"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/yaml.v2"
	"gorm.io/gorm"
	"hei-comic-api/app/httpio/out"
	"hei-comic-api/app/model"
	"hei-comic-api/app/model/mongo"
	"hei-comic-api/app/utills"
	baseMongo "hei-comic-api/base/mongo"
	"hei-comic-api/base/mysql"
	"hei-comic-api/base/redis"
	"math/rand"
	"sort"
	"sync"
	"time"
)

type BaseAdapter struct {
	Adapter
}

type Collector interface {
	InitCfg(orderId int) error                             // 初始化配置
	StartUp() error                                        //定时数据采集 脚本启动入口
	GetUpdateComics() ([]*model.Comic, error)              //获取更新列表，这样就不用全量扫描列表更新
	List(target string) ([]*model.Comic, error)            //获取漫画列表
	Details(comic *model.Comic) error                      // 漫画详情
	ChapterCount(comic *model.Comic) (int, error)          // 漫画章节总数
	Chapters(comic *model.Comic) ([]*model.Chapter, error) //获取章节列表
	GetResource(target string) ([]string, error)
	IsReferer(scope string) (bool, string)
	IsRealTime() bool

	CollectList()               //收集列表
	CollectChapter()            //收集章节
	CollectUpdateComics() error //收集更新源

	DownLoadComic(state []int) error     //文件下载，整体数据库中漫画资源下载
	ComicToNas(comic *mongo.Comic) error //单个漫画下载

	UpdateMongo(uuid int64) error
}

// NewAdapterCollector 统一实例化对象
func NewAdapterCollector(order int) Collector {
	var baseAdapter Collector
	switch order {
	case 1: //包子漫画 https://cn.baozimh.com
		baseAdapter = BaoZiAdapter
		break
	case 2: //动漫之家之漫画网 https://manhua.dmzj.com
		baseAdapter = ManHuaAdapter
		break
	case 6: //Mangabz https://www.mangabz.com
		baseAdapter = MangaBzAdapter
		break
	case 7: //http://mangastream.mobi 英文动漫
		baseAdapter = MangasTreamAdapter
		break
	case 8: //https://www.laimanhua.net 来动漫
		baseAdapter = LaiManHuaAdapter
		break
	case 9: //https://www.dushimh.com 都市漫画
		baseAdapter = DuShiAdapter
		break
	}
	return baseAdapter
}

// InitTables 初始化表 这里根据配置表中的orderId进行分表操作
func (b *BaseAdapter) InitTables(orderId int, comment string) error {
	db := mysql.NewDb()
	var err error
	comic := &model.Comic{}
	// 第一步、 检查是表是否存,创建对应的漫画表
	comicTableName := comic.GetTableName(orderId)
	err = db.Table(comicTableName).Row().Err()
	// 数据库表不存在
	if err != nil && gstr.ContainsI(err.Error(), fmt.Sprintf("%v' doesn't exist", comicTableName)) {
		err = db.Exec(comic.GetCreateTable(orderId, comment)).Error
		if err != nil {
			return err
		}
	}
	// 第二步、 创建对应的章节表
	chapter := &model.Chapter{}
	chapterTableName := chapter.GetTableName(orderId)
	err = db.Table(chapterTableName).Row().Err()
	// 数据库表不存在
	if err != nil && gstr.ContainsI(err.Error(), fmt.Sprintf("%v' doesn't exist", chapterTableName)) {
		err = db.Exec(chapter.GetCreateTable(orderId, comment)).Error
		if err != nil {
			return err
		}
	}
	return nil
}

// TongTable 每个适配器的comic表中的数据同步到mongodb数据库中。用于补齐
func (b *BaseAdapter) TongTable() {
	db := mysql.NewDb()
	mongodb := baseMongo.NewMongo().Database("comics").Collection("comics")
	var comicUUID int64
	for {
		var oldComic = make([]*model.Comic, 0, 2000)
		err := db.Table((&model.Comic{}).GetTableName(b.OrderId)).Where("uuid > ?", comicUUID).Limit(2000).Find(&oldComic).Order("uuid asc").Error
		if err != nil {
			return
		}
		if len(oldComic) == 0 {
			return
		}
		wg := sync.WaitGroup{}
		chanTask := make(chan bool, 100)
		for _, comic := range oldComic {
			wg.Add(1)
			chanTask <- true
			comicUUID = comic.UUID
			go func(comic *model.Comic) {
				defer wg.Done()
				uuid := comic.UUID
				filer := bson.D{{"uuid", uuid}}
				comicMd := mongo.Comic{}

				comicMd.Unmarshal(comic)
				update := bson.M{
					"$set": comicMd,
				}
				upsert := true
				err = mongodb.FindOneAndUpdate(context.TODO(), filer, update, &options.FindOneAndUpdateOptions{
					Upsert: &upsert,
				}).Err()
				fmt.Println(err)
				<-chanTask
			}(comic)
		}
		wg.Wait()
	}
}

// DownLoadComic 文件下载
func (b *BaseAdapter) DownLoadComic(state []int) error {
	db := mysql.NewDb()
	var comicId int64
	var err error
	for {
		comics := make([]*model.Comic, 0, 500)
		db.Table((&model.Comic{}).GetTableName(b.OrderId)).Where("state in ? and status = 2 and orderId = ? and uuid  > ?", state, b.OrderId, comicId).Limit(5000).Order("uuid asc").Find(&comics)
		if len(comics) == 0 {
			return nil
		}
		comicId, err = b.downLoadComicToNas(comics)
		if err != nil {
			return err
		}
	}
	return nil
}

// 采集漫画资源到本地无差别下载数据库中漫画
func (b *BaseAdapter) downLoadComicToNas(comics []*model.Comic) (int64, error) {
	db := mysql.NewDb()
	ctx := gctx.New()
	var comicId int64
	comicVar, err := g.Cfg().Get(ctx, "comic")
	if err != nil {
		return comicId, err
	}
	comicCfg := comicVar.MapStrVar()
	//检查目录是否存在
	//创建 漫画目录
	storageDirectory := comicCfg["storageDirectory"].String()
	sitePath := fmt.Sprintf("%v/%v", storageDirectory, b.Name)
	if !gfile.Exists(sitePath) {
		err = gfile.Mkdir(sitePath)
		if err != nil {
			return comicId, err
		}
	}
	adapter := NewAdapterCollector(b.OrderId)
	wg := sync.WaitGroup{}
	comicChan := make(chan bool, b.ComicChan)
	for _, comic := range comics {
		comicId = comic.UUID
		wg.Add(1)
		comicChan <- true
		go func(comic *model.Comic) {
			defer wg.Done()
			var (
				comicPath string
				stateStr  string
			)
			if comic.State == 2 {
				stateStr = "完结"
			}
			//开始章节下载
			comicPath = fmt.Sprintf("%v/%v_%v", sitePath, comic.UUID, utills.ReName(comic.Name))
			if !gfile.Exists(comicPath) { //老目录不存存在
				if stateStr != "" {
					comicPath = fmt.Sprintf("%v/%v/%v_%v", sitePath, stateStr, comic.UUID, utills.ReName(comic.Name))
				}
				err = gfile.Mkdir(comicPath)
				if err != nil {
					<-comicChan
					return
				}
			} else { //之前目录存在
				if stateStr != "" {
					movePath := fmt.Sprintf("%v/%v", sitePath, stateStr)
					err = gfile.Mkdir(movePath)
					if err != nil {
						<-comicChan
						return
					}
					err = gfile.Move(comicPath, fmt.Sprintf("%v/%v/%v_%v", sitePath, stateStr, comic.UUID, utills.ReName(comic.Name)))
					if err != nil {
						fmt.Println("目录移动失败")
						<-comicChan
						return
					}
					comicPath = fmt.Sprintf("%v/%v/%v_%v", sitePath, stateStr, comic.UUID, utills.ReName(comic.Name))
				}
			}
			// 下载封面
			cover := fmt.Sprintf("%v/cover.jpg", comicPath)
			if gfile.Exists(cover) && gfile.Size(cover) > 0 {

			} else {
				utills.DownloadPicture(comic.Cover, b.Referer, cover)
			}
			chapters := make([]*model.Chapter, 0)
		trySelectMysql:
			err = db.Table((&model.Chapter{}).GetTableName(b.OrderId)).Select("uuid,pid,name,resources,downloadPath").Where("pid = ? and state in ?", comic.UUID, []int{0, 1, 2}).Find(&chapters).Error
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					<-comicChan
					return
				}
				goto trySelectMysql
			}
			if len(chapters) == 0 {
				<-comicChan
				return
			}
			wgc := sync.WaitGroup{}
			chapterChan := make(chan bool, b.ChapterChan)
			for _, chapter := range chapters {
				wgc.Add(1)
				chapterChan <- true
				go func(comicPath string, chapter *model.Chapter) {
					wgc.Done()
					//检查章节是否采集过数据，未有，先去采集资源数据。再去下载
					if chapter.Resources == "" {
						b.MaxTry = 2
					tryOne:
						resourcesTmp, err := adapter.GetResource(chapter.Target)
						if err != nil || len(resourcesTmp) == 0 {
							if b.MaxTry <= 0 {
								<-chapterChan
								return
							}
							b.MaxTry--
							goto tryOne
						}
						db.Table((&model.Chapter{}).GetTableName(b.OrderId)).Where("uuid = ?", chapter.UUID).Update("resources", gstr.Join(resourcesTmp, "|"))
						chapter.Resources = gstr.Join(resourcesTmp, "|")
					}
					//创建目录
					chapterPath := fmt.Sprintf("%v/%v_%v", comicPath, chapter.UUID, utills.ReName(chapter.Name))
					if !gfile.Exists(chapterPath) {
						err = gfile.Mkdir(chapterPath)
						if err != nil {
							db.Table((&model.Chapter{}).GetTableName(b.OrderId)).Where("uuid = ?", chapter.UUID).Update("state", 2)
							<-chapterChan
							return
						}
					}
					if chapter.Resources == "" {
						db.Table((&model.Chapter{}).GetTableName(b.OrderId)).Where("uuid = ?", chapter.UUID).Update("state", 2)
						<-chapterChan
						return
					}
					//检查是否相等
					var downLoadTmp = make([]string, 0)
					if chapter.DownloadPath != "" {
						downLoadTmp = gstr.Split(chapter.DownloadPath, "|")
					}
					resources := gstr.Split(chapter.Resources, "|")
					isSkip := true
					if b.OrderId != 1 {
						if len(downLoadTmp) > 0 && len(downLoadTmp) == len(resources) { //资源对等，但是资源有损坏
							for _, v := range downLoadTmp {
								pathTmp := fmt.Sprintf("%v/%v", storageDirectory, v)
								if !gfile.Exists(pathTmp) || gfile.Size(pathTmp) <= 900 { //文件不正常
									isSkip = false
									fmt.Println("资源对等，但是资源有损坏")
									break
								}
							}
						} else {
							isSkip = false
							fmt.Println("下载过的数目：", len(downLoadTmp), "收集的数目：", len(resources))
						}
					} else {
						isSkip = false
					}

					//采集完的数据相等，并且资源下载到本地正常，跳过下面
					if isSkip {
						db.Table((&model.Chapter{}).GetTableName(b.OrderId)).Where("pid = ?", comic.UUID).Update("state", 3)
						fmt.Println("采集完的数据相等，并且资源下载到本地正常，跳过下面")
						<-chapterChan
						return
					}

					wgr := sync.WaitGroup{}
					resourceOut := make([]*out.Resource, 0, len(resources))
					resourceChan := make(chan bool, 4)
					fmt.Println(chapter.Name, "总共图片", len(resources))
					for i, resource := range resources {
						wgr.Add(1)
						resourceChan <- true
						go func(i int, resource, chapterPath string) {
							defer wgr.Done()
							var state = 1 //默认资源完整
							sort := i + 1
							b.MaxTry = 3
							fmt.Println(comic.UUID, comic.Name, "开始下载", chapter.Name, "章节的第", i, "图片")
						tryDownload:
							gfile.Remove(fmt.Sprintf("%v/%v_", comicPath, chapter.UUID))
							postfix := "jpg"
							filename := fmt.Sprintf("%v/%v.%v", chapterPath, sort, postfix)
							if gfile.Exists(filename) && gfile.Size(filename) > 900 {
								fmt.Println(comic.UUID, comic.Name, "已存在,结束下载", chapter.Name, "章节的第", i, "图片")
							} else {
								err = utills.DownloadPicture(resource, b.Referer, filename)
								if err != nil {
									if b.MaxTry <= 0 { //重试10次资源为下载下来，存在问题。
										state = 2 //章节缺失图片
									} else {
										fmt.Println("重复下载资源剩余次数", b.MaxTry)
										b.MaxTry--
										goto tryDownload
									}
								} else {
									state = 1
								}
								fmt.Println(comic.UUID, comic.Name, "结束下载", chapter.Name, "章节的第", i, "图片")
							}
							resourceOut = append(resourceOut, &out.Resource{
								Path:      filename,
								Sort:      sort,
								ChapterId: chapter.UUID,
								State:     state,
							})
							<-resourceChan
						}(i, resource, chapterPath)
					}
					wgr.Wait()
					sort.SliceStable(resourceOut, func(i, j int) bool {
						return resourceOut[i].Sort < resourceOut[j].Sort
					})
					downloadPath := make([]string, 0, len(resourceOut))

					var state = 1
					//过滤广告图片 针对包子漫画
					if b.OrderId == 1 {
						for _, v := range resourceOut {
							if gmd5.MustEncryptFile(v.Path) != "e33ef5c5e3cd1fb961b2fa65a24170bc" {
								downloadPath = append(downloadPath, gstr.ReplaceByMap(v.Path, map[string]string{
									storageDirectory + "/": "",
								}))
								if v.State == 2 {
									state = v.State
								}
							} else {
								fmt.Println("广告图片")
								gfile.Remove(v.Path)
							}
						}
					} else {
						for _, v := range resourceOut {
							downloadPath = append(downloadPath, gstr.ReplaceByMap(v.Path, map[string]string{
								storageDirectory + "/": "",
							}))
							if v.State == 2 {
								state = v.State
							}
						}
					}
					if state == 1 { // 资源检查完好，
						//循环检查资源本地化是否完整。如何全部完整。直接封版章节
						var isPerfect bool = true
						for _, pathTmp := range downloadPath {
							if !gfile.Exists(pathTmp) || gfile.Size(pathTmp) <= 900 { //文件不正常
								isPerfect = false
								fmt.Println("资源对等，但是资源有损坏")
								break
							}
						}
						//资源完美，封版章节
						if isPerfect {
							state = 3
						} else {
							state = 2 //章节部分图片损坏
						}
						db.Table((&model.Chapter{}).GetTableName(b.OrderId)).Where("uuid = ?", chapter.UUID).Updates(map[string]interface{}{
							"downloadPath": gstr.Join(downloadPath, "|"),
							"state":        state,
						})
					} else if state == 2 {
						db.Table((&model.Chapter{}).GetTableName(b.OrderId)).Where("uuid = ?", chapter.UUID).Updates(map[string]interface{}{
							"downloadPath": "",
							"state":        state,
						})
					}
					<-chapterChan
				}(comicPath, chapter)
			}
			wgc.Wait()
			db.Table((&model.Comic{}).GetTableName(b.OrderId)).Where("uuid = ?", comic.UUID).Updates(map[string]interface{}{
				"coverLocal": gstr.ReplaceByMap(cover, map[string]string{
					storageDirectory + "/": "",
				}),
			})
			//结束章节下载
			<-comicChan
		}(comic)
	}
	wg.Wait()
	return comicId, err
}

// ComicToNas 用于只保存线上上线的漫画
func (b *BaseAdapter) ComicToNas(comic *mongo.Comic) error {
	db := mysql.NewDb()
	ctx := gctx.New()
	comicVar, err := g.Cfg().Get(ctx, "comic")
	if err != nil {
		return err
	}
	comicCfg := comicVar.MapStrVar()
	//检查目录是否存在
	//创建 漫画目录
	storageDirectory := comicCfg["storageDirectory"].String()
	sitePath := fmt.Sprintf("%v/%v", storageDirectory, b.Name)
	if !gfile.Exists(sitePath) {
		err = gfile.Mkdir(sitePath)
		if err != nil {
			return err
		}
	}

	var (
		comicPath string
		stateStr  string
	)
	if comic.State == 2 {
		stateStr = "完结"
	}
	//开始章节下载
	comicPath = fmt.Sprintf("%v/%v_%v", sitePath, comic.UUID, utills.ReName(comic.Name))
	if !gfile.Exists(comicPath) { //老目录不存存在
		if stateStr != "" {
			comicPath = fmt.Sprintf("%v/%v/%v_%v", sitePath, stateStr, comic.UUID, utills.ReName(comic.Name))
		}
		err = gfile.Mkdir(comicPath)
		if err != nil {
			return err
		}
	} else { //之前目录存在
		if stateStr != "" {
			movePath := fmt.Sprintf("%v/%v", sitePath, stateStr)
			err = gfile.Mkdir(movePath)
			if err != nil {
				return err
			}
			err = gfile.Move(comicPath, fmt.Sprintf("%v/%v/%v_%v", sitePath, stateStr, comic.UUID, utills.ReName(comic.Name)))
			if err != nil {
				fmt.Println("目录移动失败")
				return err
			}
			comicPath = fmt.Sprintf("%v/%v/%v_%v", sitePath, stateStr, comic.UUID, utills.ReName(comic.Name))
		}
	}
	// 下载封面
	cover := fmt.Sprintf("%v/cover.jpg", comicPath)
	if !gfile.Exists(cover) || gfile.Size(cover) < 900 {
		utills.DownloadPicture(comic.Cover, b.Referer, cover)
	}
	chapters := make([]*model.Chapter, 0)
trySelectMysql:
	err = db.Table((&model.Chapter{}).GetTableName(b.OrderId)).Select("uuid,pid,name,resources,downloadPath").Where("pid = ? and state in ?", comic.UUID, []int{0, 1, 2}).Find(&chapters).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		goto trySelectMysql
	}
	if len(chapters) == 0 {
		return err
	}
	wgc := sync.WaitGroup{}
	chapterChan := make(chan bool, b.ChapterChan)
	adapter := NewAdapterCollector(comic.OrderId)
	for _, chapter := range chapters {
		wgc.Add(1)
		chapterChan <- true
		go func(comicPath string, chapter *model.Chapter) {
			wgc.Done()
			//检查章节是否采集过数据，未有，先去采集资源数据。再去下载
			if chapter.Resources == "" {
				b.MaxTry = 5
			tryOne:
				resourcesTmp, err := adapter.GetResource(chapter.Target)
				if err != nil || len(resourcesTmp) == 0 {
					if b.MaxTry <= 0 {
						<-chapterChan
						return
					}
					b.MaxTry--
					goto tryOne
				}
				db.Table((&model.Chapter{}).GetTableName(b.OrderId)).Where("uuid = ?", chapter.UUID).Update("resources", gstr.Join(resourcesTmp, "|"))
				chapter.Resources = gstr.Join(resourcesTmp, "|")
			}
			//创建目录
			chapterPath := fmt.Sprintf("%v/%v_%v", comicPath, chapter.UUID, utills.ReName(chapter.Name))
			if !gfile.Exists(chapterPath) {
				err = gfile.Mkdir(chapterPath)
				if err != nil {
					db.Table((&model.Chapter{}).GetTableName(b.OrderId)).Where("uuid = ?", chapter.UUID).Update("state", 2)
					<-chapterChan
					return
				}
			}
			if chapter.Resources == "" {
				db.Table((&model.Chapter{}).GetTableName(b.OrderId)).Where("uuid = ?", chapter.UUID).Update("state", 2)
				<-chapterChan
				return
			}
			//检查是否相等
			var downLoadTmp = make([]string, 0)
			if chapter.DownloadPath != "" {
				downLoadTmp = gstr.Split(chapter.DownloadPath, "|")
			}
			resources := gstr.Split(chapter.Resources, "|")
			isSkip := true
			if len(downLoadTmp) > 0 && len(downLoadTmp) == len(resources) { //资源对等，但是资源有损坏
				for _, v := range downLoadTmp {
					pathTmp := fmt.Sprintf("%v/%v", storageDirectory, v)
					if !gfile.Exists(pathTmp) || gfile.Size(pathTmp) <= 900 { //文件不正常
						isSkip = false
						fmt.Println("资源对等，但是资源有损坏")
						break
					}
				}
			} else {
				isSkip = false
				fmt.Println("下载过的数目：", len(downLoadTmp), "收集的数目：", len(resources))
			}
			//采集完的数据相等，并且资源下载到本地正常，跳过下面
			if isSkip {
				db.Table((&model.Chapter{}).GetTableName(b.OrderId)).Where("pid = ?", comic.UUID).Update("state", 3)
				fmt.Println("采集完的数据相等，并且资源下载到本地正常，跳过下面")
				<-chapterChan
				return
			}
			wgr := sync.WaitGroup{}
			resourceOut := make([]*out.Resource, 0, len(resources))
			resourceChan := make(chan bool, 4)
			fmt.Println(chapter.Name, "总共图片", len(resources))
			for i, resource := range resources {
				wgr.Add(1)
				resourceChan <- true
				go func(i int, resource, chapterPath string) {
					var state = 1 //默认资源完整
					defer wgr.Done()
					sort := i + 1
					b.MaxTry = 3
					fmt.Println(comic.UUID, comic.Name, "开始下载", chapter.Name, "章节的第", i, "图片")
				tryDownload:
					gfile.Remove(fmt.Sprintf("%v/%v_", comicPath, chapter.UUID))
					postfix := "jpg"
					filename := fmt.Sprintf("%v/%v.%v", chapterPath, sort, postfix)
					if gfile.Exists(filename) && gfile.Size(filename) > 900 {
						fmt.Println(comic.UUID, comic.Name, "已存在,结束下载", chapter.Name, "章节的第", i, "图片")
					} else {
						err = utills.DownloadPicture(resource, b.Referer, filename)
						if err != nil {
							if b.MaxTry <= 0 { //重试10次资源为下载下来，存在问题。
								state = 2 //章节缺失图片
							} else {
								fmt.Println("重复下载资源剩余次数", b.MaxTry)
								b.MaxTry--
								goto tryDownload
							}
						} else {
							state = 1 //资源正常
						}
						fmt.Println(comic.UUID, comic.Name, "结束下载", chapter.Name, "章节的第", i, "图片")
					}
					resourceOut = append(resourceOut, &out.Resource{
						Path:      filename,
						Sort:      sort,
						ChapterId: chapter.UUID,
						State:     state,
					})
					<-resourceChan
				}(i, resource, chapterPath)
			}
			wgr.Wait()
			sort.SliceStable(resourceOut, func(i, j int) bool {
				return resourceOut[i].Sort < resourceOut[j].Sort
			})
			downloadPath := make([]string, 0, len(resourceOut))
			//外面检查一下是否有损坏是
			var state = 1
			//过滤广告图片 针对包子漫画
			if b.OrderId == 1 {
				for _, v := range resourceOut {
					if gmd5.MustEncryptFile(v.Path) != "e33ef5c5e3cd1fb961b2fa65a24170bc" {
						downloadPath = append(downloadPath, gstr.ReplaceByMap(v.Path, map[string]string{
							storageDirectory + "/": "",
						}))
						if v.State == 2 {
							state = v.State
						}
					} else {
						fmt.Println("广告图片")
						gfile.Remove(v.Path)
					}
				}
			} else {
				for _, v := range resourceOut {
					downloadPath = append(downloadPath, gstr.ReplaceByMap(v.Path, map[string]string{
						storageDirectory + "/": "",
					}))
					if v.State == 2 {
						state = v.State
					}
				}
			}

			if state == 1 { // 资源检查完好，
				//循环检查资源本地化是否完整。如何全部完整。直接封版章节
				var isPerfect bool = true
				for _, pathTmp := range downloadPath {
					if !gfile.Exists(pathTmp) || gfile.Size(pathTmp) <= 900 { //文件不正常
						isPerfect = false
						fmt.Println("资源对等，但是资源有损坏")
						break
					}
				}
				//资源完美，封版章节
				if isPerfect {
					state = 3
				} else {
					state = 2 //章节部分图片损坏
				}
				db.Table((&model.Chapter{}).GetTableName(b.OrderId)).Where("uuid = ?", chapter.UUID).Updates(map[string]interface{}{
					"downloadPath": gstr.Join(downloadPath, "|"),
					"state":        state,
				})
			} else if state == 2 {
				db.Table((&model.Chapter{}).GetTableName(b.OrderId)).Where("uuid = ?", chapter.UUID).Updates(map[string]interface{}{
					"downloadPath": "",
					"state":        state,
				})
			}
			<-chapterChan
		}(comicPath, chapter)
	}
	wgc.Wait()
	db.Table((&model.Comic{}).GetTableName(b.OrderId)).Where("uuid = ?", comic.UUID).Updates(map[string]interface{}{
		"coverLocal": gstr.ReplaceByMap(cover, map[string]string{
			storageDirectory + "/": "",
		}),
	})

	return err
}

// InitCfg 初始化配置
func (b *BaseAdapter) InitCfg(orderId int) error {
	ctx := gctx.New()
	adapterVar, err := g.Cfg().Get(ctx, "adapter")
	if err != nil {
		return err
	}
	adapterVars := adapterVar.Vars()
	for _, cfgVar := range adapterVars {
		adapter := Adapter{}
		err = yaml.Unmarshal(cfgVar.Bytes(), &adapter)
		if err != nil {
			return err
		}
		if adapter.OrderId == orderId {
			fmt.Println(adapter.Name, adapter.OrderId, orderId)
			b.Adapter = adapter
			return nil
		}
	}
	return nil
}

// IsReferer 是否开启防盗链接
func (b *BaseAdapter) IsReferer(scope string) (bool, string) {
	if b.Referer != "" { //是否开启防盗链接破解
		if scope == "" {
			return true, b.Referer
		}
		if gstr.InArray(b.Scope, scope) {
			return true, b.Referer
		}
	}
	return false, b.Referer
}

func (b *BaseAdapter) IsRealTime() bool {
	return b.RealTime
}

// CollectList 采集列表,并且补齐详情，初始化网站数据使用或者手动补齐数据使用
func (b *BaseAdapter) CollectList() {
	fmt.Println(fmt.Sprintf("%v开始解析%v网站数据", b.Name, b.Origin))
	// 第一步、 先收集列表数据
	db := mysql.NewDb()
	ctx := context.TODO()
	wgL := sync.WaitGroup{}
	pageChan := make(chan bool, 2)
	redisClient := redis.NewRedis()

	for _, targetTpl := range b.ListTemplate {
		wgL.Add(1)
		pageChan <- true
		go func(targetTpl string) {
			defer wgL.Done()
			var page int = 1
			var err error
			var pageKey string
			pageKey = fmt.Sprintf("page_%v_%v", gmd5.MustEncryptString(targetTpl), b.OrderId)
			if b.IsPage { //需要分页采集
				page, err = redisClient.Get(ctx, pageKey).Int()
				if err != nil {
					page = 1
				}
			}
			page = 1
			for {
				var target string
				if b.IsPage { //需要分页采集
					count := gstr.Count(targetTpl, "%v")
					if count == 1 {
						target = fmt.Sprintf(targetTpl, page)
					} else {
						target = fmt.Sprintf(targetTpl, b.Origin, page)
					}
				} else { //不需要分页采集
					target = fmt.Sprintf(targetTpl, b.Origin)
				}
				isNext := b.makeComicList(db, target)
				if !isNext {
					if b.MaxTry <= 0 {
						fmt.Println(b.Name, "列表数据已经空了", page)
						break
					}
					b.MaxTry--
				} else {
					b.MaxTry = 5 //成功一次，重置之前的数目
					fmt.Println("成功--处理", targetTpl, "列表的第", page, "页")
				}
				if b.IsPage {
					page++
					redisClient.Set(ctx, pageKey, page, 1*time.Hour)
				} else {
					fmt.Println(b.Name, "无需分页，一次跑完", page)
					break
				}
			}
			<-pageChan
			redisClient.Del(ctx, pageKey)
		}(targetTpl)
	}
	wgL.Wait()
	fmt.Println(fmt.Sprintf("结束解析%v网站数据", b.Origin))
}

// CollectUpdateComics 按照时间去网站拉取最新数据，并且补齐章节信息，同时校验章节损坏程度，自动下架资源
func (b *BaseAdapter) CollectUpdateComics() error {
	fmt.Println("执行更新数据收集", b.Name)
	db := mysql.NewDb()
	redis := redis.NewRedis()
	startTime := time.Now().Unix()
	adapter := NewAdapterCollector(b.OrderId)
	comics, err := adapter.GetUpdateComics()
	if err != nil {
		return err
	}
	if len(comics) == 0 {
		return nil
	}
	comicSet := gset.New(true)
	wgComic := sync.WaitGroup{}
	comicChan := make(chan bool, b.ComicChan)
	for _, comic := range comics {
		wgComic.Add(1)
		comicChan <- true
		go func(comic *model.Comic) {
			defer wgComic.Done()
			var comicId int64
			if !comicSet.Contains(comic.Target) {
				isLock, err := redis.SetNX(context.TODO(), gmd5.MustEncryptString(comic.Target), 1, 3*time.Hour).Result()
				if err != nil {
					return
				}
				if !isLock {
					<-comicChan
					fmt.Println(comic.Name, "漫画3小时之前收集过了，略过。。。。")
					return
				}
				data := &model.Comic{}
				err = db.Table((&model.Comic{}).GetTableName(b.OrderId)).Where("target = ?", comic.Target).First(data).Error
				if err != nil {
					if !errors.Is(err, gorm.ErrRecordNotFound) {
						<-comicChan
						return
					}
				}
				if data.UUID == 0 {
					err = adapter.Details(comic)
					if err != nil {
						fmt.Println("错误信息", err)
						<-comicChan
						return
					}
					count, _ := adapter.ChapterCount(comic)
					if count > 0 {
						err = db.Table((&model.Comic{}).GetTableName(b.OrderId)).Create(comic).Error
						if err != nil {
							fmt.Println("错误信息", err)
							<-comicChan
							return
						}
						fmt.Println(comic.UUID, "新增漫画-附加详情", comic.Name)
						// 采集章节信息
						err = b.makeChapters(db, adapter, comic)
						if err != nil {
							<-comicChan
							return
						}
						// 章节写入数据库
						comicId = comic.UUID
						fmt.Println(comic.UUID, "新增漫画-章节数据补齐成功", data.Name)
					}
				} else { //数据存在
					if true { //因为详情会变化，所有要去检查一下
						if data.State != 2 {
							err = adapter.Details(comic)
							if err != nil {
								fmt.Println("错误信息", err)
								<-comicChan
								return
							}
							if data.State != comic.State {
								err = db.Table((&model.Comic{}).GetTableName(b.OrderId)).Where("uuid = ?", data.UUID).Updates(comic).Error
								if err != nil {
									fmt.Println("错误信息", err)
									<-comicChan
									return
								}
							}
						}
						// 采集章节信息
						err = b.makeChapters(db, adapter, data)
						if err != nil {
							<-comicChan
							return
						}

						// 章节写入数据库
						db.Table((&model.Comic{}).GetTableName(b.OrderId)).Where("uuid = ?", data.UUID).Update("updateTime", time.Now().Unix())
						comicId = data.UUID
						fmt.Println(data.UUID, "更新漫画-章节数据补齐成功", data.Name)
					}
				}
				comicSet.Add(comic.Target)
			}
			//检查漫画资源是否可用上线
			if comicId > 0 {
				b.UpdateMongo(comicId)
			}
			<-comicChan
		}(comic)
	}
	wgComic.Wait()
	comicSet.Clear()
	// 核心处理 结束
	fmt.Println("执行更新数据补齐成功", time.Now().Unix()-startTime, "秒")
	return err
}

// CollectChapter 收集章节资源数据  补齐校验章节数据，手动执行
func (b *BaseAdapter) CollectChapter() {
	db := mysql.NewDb()
	adapter := NewAdapterCollector(b.OrderId)
	var comicId int64
	startTime := utills.GetYesterdayDayTimeUnix(10) //2天之前
	endTime := time.Now().Unix()
	for {
		var err error
		comics := make([]*model.Comic, 0, 500)
		//第一步、 查询漫画章节采集完毕的数据
		err = db.Table((&model.Comic{}).GetTableName(b.OrderId)).
			Where("uuid > ?", comicId).
			Where("updateTime BETWEEN ? AND ?", startTime, endTime).
			Limit(5000).Find(&comics).Error

		if err != nil {
			continue
		}
		if len(comics) == 0 {
			return
		}
		wgComic := sync.WaitGroup{}
		comicChan := make(chan bool, b.ComicChan)
		for _, comic := range comics {
			comicId = comic.UUID
			wgComic.Add(1)
			comicChan <- true
			go func(comic *model.Comic) {
				defer wgComic.Done()
				chapters := make([]*model.Chapter, 0, 1000)
				err = db.Table((&model.Chapter{}).GetTableName(b.OrderId)).
					Where("pid = ? and state in ?", comic.UUID, []int{0}).Find(&chapters).Error
				if err != nil {
					<-comicChan
					return
				}
				wg := sync.WaitGroup{}
				chanTask := make(chan bool, b.ChapterChan)
				for _, chapter := range chapters {
					wg.Add(1)
					chanTask <- true
					go func(chapter *model.Chapter) {
						defer wg.Done()
						var state int = 1
						if chapter.Resources == "" {
							b.MaxTry = 1
						tryResource:
							resource, err := adapter.GetResource(chapter.Target)
							if err == nil { // 没有任何报错
								if len(resource) > 0 {
									rand.Seed(time.Now().UnixNano())
									num := rand.Intn(len(resource))
									imgUrl := resource[num]
									if !utills.CheckResource(imgUrl, comic.Origin) {
										state = 2
									}
									db.Table((&model.Chapter{}).GetTableName(b.OrderId)).Where("uuid = ?", chapter.UUID).Updates(map[string]interface{}{
										"resources": gstr.Join(resource, "|"),
										"state":     state,
									})
								} else {
									if b.MaxTry <= 0 {
										db.Table((&model.Chapter{}).GetTableName(b.OrderId)).Where("uuid = ?", chapter.UUID).Updates(map[string]interface{}{
											"resources": gstr.Join(resource, "|"),
											"state":     2,
										})
										fmt.Println("资源不可用")
										<-chanTask
										return
									} else {
										b.MaxTry--
										fmt.Println(b.MaxTry)
										goto tryResource
									}
								}
							} else {
								db.Table((&model.Chapter{}).GetTableName(b.OrderId)).Where("uuid = ?", chapter.UUID).Updates(map[string]interface{}{
									"resources": gstr.Join(resource, "|"),
									"state":     2,
								})
							}
						} else { // 资源采集过了
							var imgUrl string
							resource := gstr.Split(chapter.Resources, "|")
							rand.Seed(time.Now().UnixNano())
							num := rand.Intn(len(resource))
							imgUrl = resource[num]
							if !utills.CheckResource(imgUrl, comic.Origin) {
								state = 2
							}
							if state != chapter.State {
								db.Table((&model.Chapter{}).GetTableName(comic.OrderId)).Where("uuid = ?", chapter.UUID).Updates(map[string]interface{}{
									"resources": gstr.Join(resource, "|"),
									"state":     state,
								})
							}
						}
						<-chanTask
					}(chapter)
				}
				wg.Wait()
				<-comicChan
				//章节采集完毕
			}(comic)
		}
		wgComic.Wait()
	}
}

// 统一换采集 漫画列表 列表
func (b *BaseAdapter) makeComicList(db *gorm.DB, target string) bool {
	var err error
	comicSet := gset.New(true)
	adapter := NewAdapterCollector(b.OrderId)
	// 核心处理 开始
	comics, err := adapter.List(target)
	if err != nil {
		return false
	}
	if len(comics) == 0 {
		return false
	}
	wgComic := sync.WaitGroup{}
	comicChan := make(chan bool, 20000)
	for _, comic := range comics {
		wgComic.Add(1)
		comicChan <- true
		go func(comic *model.Comic) {
			defer wgComic.Done()
			if !comicSet.Contains(comic.Target) {
				data := &model.Comic{}
				err = db.Table((&model.Comic{}).GetTableName(b.OrderId)).Where("target = ?", comic.Target).First(data).Error
				if err != nil {
					if !errors.Is(err, gorm.ErrRecordNotFound) {
						<-comicChan
						return
					}
				}
				if data.UUID == 0 {
					err = adapter.Details(comic)
					if err != nil {
						fmt.Println("错误信息", err)
						<-comicChan
						return
					}
					count, _ := adapter.ChapterCount(comic)
					if count > 0 {
						err = db.Table((&model.Comic{}).GetTableName(b.OrderId)).Create(comic).Error
						if err != nil {
							fmt.Println("错误信息", err)
							<-comicChan
							return
						}
						fmt.Println(comic.UUID, "新增漫画-附加详情", comic.Name)
						// 采集章节信息
						err = b.makeChapters(db, adapter, comic)
						if err != nil {
							fmt.Println("错误信息", err)
							<-comicChan
							return
						}
						err = b.UpdateMongo(comic.UUID)
						if err != nil {
							<-comicChan
							fmt.Println(comic.UUID, "新增漫画-mongo数据补齐失败", comic.Name)
							return
						}
						fmt.Println(comic.UUID, "新增漫画-章节数据补齐成功", data.Name)
					}
				} else { //数据存在
					if true { //因为详情会变化，所有要去检查一下
						if data.State != 2 || true {
							err = adapter.Details(comic)
							if err != nil {
								fmt.Println("错误信息", err)
								<-comicChan
								return
							}
							if data.State != comic.State {
								err = db.Table((&model.Comic{}).GetTableName(b.OrderId)).Where("uuid = ?", data.UUID).Updates(comic).Error
								if err != nil {
									fmt.Println("错误信息", err)
									<-comicChan
									return
								}
							}
						}

						// 采集章节信息
						err = b.makeChapters(db, adapter, data)
						if err != nil {
							<-comicChan
							fmt.Println(data.Name, "错误信息", err)
							return
						}
						// 写入数据库
						db.Table((&model.Comic{}).GetTableName(b.OrderId)).Where("uuid = ?", data.UUID).Update("updateTime", time.Now().Unix())
						err = b.UpdateMongo(data.UUID)
						if err != nil {
							fmt.Println(data.UUID, "更新漫画-mongo数据补齐失败", data.Name)
						}
						fmt.Println(data.UUID, "更新漫画-章节数据补齐成功", data.Name)
					}
				}
				//最终评估一下资源是否可用状态，是否可用上架漫画到mongod数据库
				comicSet.Add(comic.Target)
			}
			<-comicChan
		}(comic)
	}
	wgComic.Wait()
	fmt.Println("断点测试")
	comicSet.Clear()
	// 核心处理 结束
	return true
}

func (b *BaseAdapter) makeChapters(db *gorm.DB, adapter Collector, comic *model.Comic) error {
	var err error
	comicUUID := comic.UUID //漫画主键
	b.MaxTry = 5
stepChapter:
	chapters, err := adapter.Chapters(comic)
	if len(chapters) == 0 {
		return errors.New("采集章节为空")
	}
	if err != nil { //
		if b.MaxTry <= 0 {
			return errors.New(fmt.Sprintf("%v,%v,章节数据采集失败", comicUUID, comic.Name))
		}
		b.MaxTry--
		goto stepChapter
	}

	//查看章节列表是否存在数据库
	chaptersModel := make([]*model.Chapter, 0)
	err = db.Table((&model.Chapter{}).GetTableName(b.OrderId)).Where("pid = ?", comicUUID).Find(&chaptersModel).Error
	fmt.Println(comic.Name, "执行到", len(chaptersModel))
	//转换map
	chapterMap := make(map[string]*model.Chapter)
	for _, chapter := range chaptersModel {
		chapterMap[chapter.Target] = chapter
	}

	//批量保存章节数据
	chapterSet := gset.New(true)
	bitchChapters := make([]*model.Chapter, 0)
	wgC := sync.WaitGroup{}
	chapterChan := make(chan bool, 10)
	rand.Seed(time.Now().UnixNano())
	for _, chapter := range chapters {
		wgC.Add(1)
		chapterChan <- true
		go func(chapter *model.Chapter) {
			defer wgC.Done()
			//查询章节是否存在
			_, ok := chapterMap[chapter.Target]
			if !ok {
				//自身过滤重复
				if !chapterSet.Contains(chapter.Target) {
					chapterSet.Add(chapter.Target)
					//并且采集章节数据
					if comic.Status == 2 {
						fmt.Println(comic.Name, chapter.Name, chapter.Target)
						resources, _ := adapter.GetResource(chapter.Target)
						if len(resources) > 0 {
							num := rand.Intn(len(resources))
							imgUrl := resources[num]
							if !utills.CheckResource(imgUrl, comic.Origin) {
								chapter.State = 2
							} else {
								chapter.State = 1
							}
							chapter.Resources = gstr.Join(resources, "|")
						} else {
							chapter.State = 2
						}
					}
					bitchChapters = append(bitchChapters, chapter)
				}
			}
			<-chapterChan
		}(chapter)
	}
	wgC.Wait()
	chapterSet.Clear()
	//二次保险，再次过滤一下
	chapterTwSet := gset.New(false)
	endBitchChapters := make([]*model.Chapter, 0)
	for _, chapter := range bitchChapters {
		if !chapterTwSet.Contains(chapter.Target) {
			chapterTwSet.Add(chapter.Target)
			endBitchChapters = append(endBitchChapters, chapter)
		} else {
			fmt.Println("数据存在，忽略", chapter.Target, chapter.Name)
		}
	}
	chapterTwSet.Clear()
	if len(endBitchChapters) == 0 {
		return errors.New("未有章节数据更新")
	}

	//批量插入数据量
	err = db.Table((&model.Chapter{}).GetTableName(b.OrderId)).CreateInBatches(endBitchChapters, len(endBitchChapters)).Error
	if err != nil {
		errString := fmt.Sprintf("%v,%v漫画,新增章节数据失败", comic.UUID, comic.Name)
		return errors.New(errString)
	}
	return nil
}

// UpdateMongo 补齐mongod 数据信息
func (b *BaseAdapter) UpdateMongo(uuid int64) error {
	db := mysql.NewDb()
	comic := &model.Comic{}
	err := db.Table((&model.Comic{}).GetTableName(b.OrderId)).Where("uuid = ?", uuid).First(&comic).Error
	if err != nil {
		return err
	}

	mongodb := baseMongo.NewMongo().Database("comics").Collection("comics")
	filer := bson.D{{"uuid", uuid}}
	comicMd := mongo.Comic{}
	comicMd.Unmarshal(comic)
	update := bson.M{
		"$set": comicMd,
	}
	upsert := true
	err = mongodb.FindOneAndUpdate(context.TODO(), filer, update, &options.FindOneAndUpdateOptions{
		Upsert: &upsert,
	}).Err()
	if err != nil {
		if err.Error() == "mongo: no documents in result" {
			return nil
		}
		return err
	}
	return nil
}

// 修复封面
func (b *BaseAdapter) FixComicCover() {
	db := mysql.NewDb()
	var comicUUID int64
	for {
		var oldComic = make([]*model.Comic, 0, 2000)
		err := db.Table((&model.Comic{}).GetTableName(b.OrderId)).Where("uuid > ? and status = 0", comicUUID).Limit(2000).Find(&oldComic).Order("uuid asc").Error
		if err != nil {
			return
		}
		if len(oldComic) == 0 {
			return
		}
		wg := sync.WaitGroup{}
		chanTask := make(chan bool, 100)
		for _, comic := range oldComic {
			wg.Add(1)
			chanTask <- true
			comicUUID = comic.UUID
			go func(comic *model.Comic) {
				defer wg.Done()
				//检查封面是否可用
				ctx := context.TODO()
				client := g.Client()
				client.SetHeader("Referer", b.Referer)
				rsp, err := client.Get(ctx, comic.Cover)
				defer rsp.Close()
				if err != nil {
					<-chanTask
					return
				}
				if rsp.ContentLength > 0 {
					fmt.Println("封面正常")
					<-chanTask
					return
				}
				//以包子漫画资源为标准，如何没有只能这样了。后期递归补全
				comicM := &model.Comic{}
				db.Table((&model.Comic{}).GetTableName(1)).Where("name = ?", comic.Name).First(&comicM)
				if comicM.UUID > 0 {
					db.Table((&model.Comic{}).GetTableName(b.OrderId)).Where("uuid = ?", comic.UUID).Update("cover", comicM.Cover)
					b.UpdateMongo(comic.UUID)
				}
				<-chanTask
			}(comic)
		}
		wg.Wait()
	}
}
