package crond

import (
	"context"
	"errors"
	"fmt"
	"github.com/gogf/gf/v2/container/gset"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/text/gstr"
	"go.mongodb.org/mongo-driver/bson"
	mongo2 "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/gorm"
	"hei-comic-api/app/collector/adapter"
	"hei-comic-api/app/model"
	"hei-comic-api/app/model/mongo"
	"hei-comic-api/app/utills"
	baseMongo "hei-comic-api/base/mongo"
	"hei-comic-api/base/mysql"
	"math/rand"
	"sort"
	"sync"
	"time"
)

var OnShelfComicCrond = new(onShelfComics)

type onShelfComics struct{}

type comparComicName struct {
	ComicNA *ComicNA `bson:"_id"`
}

type ComicNA struct {
	Name   string
	Author string
}

type comicSort struct {
	Sort  int
	Comic *mongo.Comic
}

type comparisonChapter struct {
	Total           int              `json:"total"`
	TotalChapters   []*model.Chapter `json:"totalChapters"`
	Normal          int              `json:"normal"`
	NormalChapters  []*model.Chapter `json:"normalChapters"`
	Damage          int              `json:"damage"`
	DamageChapters  []*model.Chapter `json:"damageChapters"`
	NotInit         int              `json:"notInit"`
	NotInitChapters []*model.Chapter `json:"notInitChapters"`
}

// SealedComics 封版漫画。 定时扫描完结漫画资源是否完整并且是否已经保存到本地。然后全部通过，直接封版资源。
func (c *onShelfComics) SealedComics() error {
	filter := make(bson.M)
	filter["status"] = 2
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
			TotalChapters := make([]*model.Chapter, 0)
			err = db.Table((&model.Chapter{}).GetTableName(orderId)).Where("pid = ? and state = 3", uuid).Find(&TotalChapters).Error
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
						resources, _ := adapter.NewAdapterCollector(orderId).GetResource(chapter.Target)
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
	//结束 检查数据是否可用封版
	return nil
}

// ScanComics 扫描漫画数据，检查是否完整，
func (c *onShelfComics) ScanComics() {
	//@TODO::海外英文漫画，直接上线，因为现在只有这一家
	c.updateEnComic(7, 2)
	ctx := context.TODO()
	sortMap := make(map[int]int)
	mongoClient := baseMongo.NewMongo().Database("comics").Collection("comics")
	//排序一下漫画资源。 漫画根据配置中的顺序进行排序
	adapterVar, err := g.Cfg().Get(ctx, "adapter")
	if err != nil {
		return
	}
	adaptersCfg := adapterVar.Vars()
	// 数据资源优先原则，顺序采集
	for _, adapterCfg := range adaptersCfg {
		adapterMap := adapterCfg.MapStrVar()
		sortMap[adapterMap["orderId"].Int()] = adapterMap["sort"].Int()
	}
	//开始  分组查询数据
	comicNames := make([]*comparComicName, 0)
	groupFilter := make(bson.M, 0)
	groupFilter["status"] = 0
	groupFilter["orderId"] = bson.M{
		"$in": []int{
			1, 2, 6, 8, 9,
		},
	}
	//offset := (page - 1) * limit
	pipeline := mongo2.Pipeline{
		bson.D{ //排序
			{"$sort", bson.D{{"name", -1}}},
		},
		bson.D{ //查询条件
			{"$match", groupFilter},
		},
		//bson.D{
		//	{"$limit", limit},
		//},
		//bson.D{
		//	{"$skip", offset},
		//},
		bson.D{
			{
				"$group", bson.D{
					{"_id", bson.D{
						{"name", "$name"},
					}},
				},
			},
		},
	}
	opts := options.Aggregate()
	cursor, err := mongoClient.Aggregate(ctx, pipeline, opts)
	if cursor == nil {
		fmt.Println("没有数据可以执行了")
		return
	}
	defer cursor.Close(ctx)
	if err = cursor.All(ctx, &comicNames); err != nil {
		return
	}
	if len(comicNames) == 0 {
		fmt.Println("漫画数据：", len(comicNames), "没有数据可以执行了")
		return
	}
	//结束  分组查询数据

	// 开始处理 后续数据
	wg := sync.WaitGroup{}
	taskChan := make(chan bool, 50)
	for _, comicName := range comicNames {
		wg.Add(1)
		taskChan <- true
		go func(comicName *comparComicName) {
			defer wg.Done()
			fmt.Println("开始对比漫画数据:", comicName.ComicNA.Name)
			if err = c.compareComics(mongoClient, comicName.ComicNA, sortMap); err != nil {
				fmt.Println("扫描漫画数据完整功能,scanComics->错误信息", err)
			}
			<-taskChan
		}(comicName)
	}
	wg.Wait()
}

// 查询已有数据
func (c *onShelfComics) compareComics(mongoClient *mongo2.Collection, comicNA *ComicNA, sortMap map[int]int) error {
	ctx := context.TODO()
	db := mysql.NewDb()
	bestComic := &mongo.Comic{}
	spareComic := make([]*mongo.Comic, 0)
	comics := make([]*mongo.Comic, 0, 21)
	var findOpts = &options.FindOptions{}
	sorts := bson.D{{"orderId", 1}}
	findOpts.SetSort(sorts)
	cur, err := mongoClient.Find(ctx, bson.M{"name": comicNA.Name}, findOpts)
	if err != nil {
		return err
	}
	defer cur.Close(ctx)
	err = cur.All(ctx, &comics)
	if err != nil {
		return err
	}
	var isHasBestComic bool
	//排序漫画
	comicSorts := make([]*comicSort, 0)
	for _, comic := range comics {
		comicSorts = append(comicSorts, &comicSort{
			Sort:  sortMap[comic.OrderId],
			Comic: comic,
		})
	}
	//排序一下
	sort.Slice(comicSorts, func(i, j int) bool {
		return comicSorts[i].Sort < comicSorts[j].Sort
	})
	comicNew := make([]*mongo.Comic, 0)
	for _, comic := range comicSorts {
		comicNew = append(comicNew, comic.Comic)
	}

	//检查是否存在最佳上架漫画
	for _, comic := range comicNew {
		if comic.Status == 2 {
			bestComic = comic
			isHasBestComic = true
			break
		}
	}

	// 之前没有上架过最佳数据，直接去对比找到最佳数据
	var bestStatus int = 2
	if !isHasBestComic {
		for _, comic := range comicNew {
			//包子漫画 最好的，直接采纳
			if comic.OrderId == 1 { //出现这个，其他资源不可用，只有包子资源默认可用度更高
				bestComic = comic
				isHasBestComic = true
				bestStatus = 5
				break
			}
			//先补齐一下章节数据
			tryOne := 2
		polish:
			comicM := comic.MarshalToModelComic()
			err = c.polishChapters(db, comicM)
			if err != nil {
				fmt.Println("补齐章节数据失败")
				if tryOne > 0 {
					tryOne--
					goto polish
				}
			}
			chapter := c.getComicChapters(db, comic.UUID, comic.OrderId)
			if chapter.Total > 0 && chapter.Normal == chapter.Total { //章节全部是最完美的，如果不存在最完美的章节，需要手动筛选
				bestComic = comic
				isHasBestComic = true
				break
			}
		}
	}
	// 处理之前存在最佳漫画，之后其他源又收录过来，直接让新源变成后备源
	for _, comic := range comicNew {
		//查询到最佳的，启用放入备用
		if bestComic.UUID != comic.UUID {
			spareComic = append(spareComic, comic)
		}
	}
	//更新最佳数据
	if bestComic.UUID > 0 {
		Common.updateComic(bestComic.UUID, bestComic.OrderId, bestStatus)
	}
	var status int
	if isHasBestComic {
		status = 3 //只是备用
	} else {
		status = 4 //没有找到最优漫画资源，需要手动提取(包子资源也没有)
	}
	//更新备用数据状态
	for _, comic := range spareComic {
		Common.updateComic(comic.UUID, comic.OrderId, status)
	}
	return nil
}

func (c *onShelfComics) getComicChapters(db *gorm.DB, uuid int64, orderId int) *comparisonChapter {
	comparisonChapterData := &comparisonChapter{}
	var (
		TotalChapters   = make([]*model.Chapter, 0)
		NormalChapters  = make([]*model.Chapter, 0)
		DamageChapters  = make([]*model.Chapter, 0)
		NotInitChapters = make([]*model.Chapter, 0)
	)
	err := db.Table((&model.Chapter{}).GetTableName(orderId)).Where("pid = ?", uuid).Find(&TotalChapters).Error
	if err != nil {
		return comparisonChapterData
	}
	// 开启协程采集章节
	wg := sync.WaitGroup{}
	taskChan := make(chan bool, 3)
	for _, chapter := range TotalChapters {
		wg.Add(1)
		taskChan <- true
		go func(chapter *model.Chapter) {
			defer wg.Done()
			if chapter.State == 0 || chapter.Resources == "" {
				resources, _ := adapter.NewAdapterCollector(orderId).GetResource(chapter.Target)
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
			<-taskChan
		}(chapter)
	}
	wg.Wait()

	for _, chapter := range TotalChapters {
		if chapter.State == 1 {
			NormalChapters = append(NormalChapters, chapter)
		} else if chapter.State == 2 {
			DamageChapters = append(DamageChapters, chapter)
		} else if chapter.State == 0 {
			NotInitChapters = append(NotInitChapters, chapter)
		}
	}
	comparisonChapterData.Total = len(TotalChapters)
	comparisonChapterData.Normal = len(NormalChapters)
	comparisonChapterData.Damage = len(DamageChapters)
	comparisonChapterData.TotalChapters = TotalChapters
	comparisonChapterData.NormalChapters = NormalChapters
	comparisonChapterData.DamageChapters = DamageChapters
	comparisonChapterData.NotInit = len(NotInitChapters)
	comparisonChapterData.NotInitChapters = NotInitChapters

	return comparisonChapterData
}

func (c *onShelfComics) polishChapters(db *gorm.DB, comic *model.Comic) error {
	var err error
	adapter := adapter.NewAdapterCollector(comic.OrderId)
	comicUUID := comic.UUID //漫画主键
	maxTry := 5
stepChapter:
	chapters, err := adapter.Chapters(comic)
	if len(chapters) == 0 {
		return errors.New("采集章节为空")
	}
	if err != nil { //
		if maxTry <= 0 {
			return errors.New(fmt.Sprintf("%v,%v,章节数据采集失败", comicUUID, comic.Name))
		}
		maxTry--
		goto stepChapter
	}

	//查看章节列表是否存在数据库
	chaptersModel := make([]*model.Chapter, 0)
	err = db.Table((&model.Chapter{}).GetTableName(comic.OrderId)).Where("pid = ?", comicUUID).Find(&chaptersModel).Error
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
		return nil
	}

	//批量插入数据量
	err = db.Table((&model.Chapter{}).GetTableName(comic.OrderId)).CreateInBatches(endBitchChapters, len(endBitchChapters)).Error
	if err != nil {
		errString := fmt.Sprintf("%v,%v漫画,新增章节数据失败", comic.UUID, comic.Name)
		return errors.New(errString)
	}
	return nil
}

func (c *onShelfComics) updateEnComic(orderId int, status int) {
	db := mysql.NewDb()
	mongoClient := baseMongo.NewMongo().Database("comics").Collection("comics")
	db.Table((&model.Comic{}).GetTableName(orderId)).Where("orderId = ?", orderId).Update("status", status)
	filer := bson.D{{"orderId", orderId}}
	update := bson.M{
		"$set": bson.M{
			"status": status,
		},
	}
	upsert := false
	mongoClient.UpdateMany(context.TODO(), filer, update, &options.UpdateOptions{
		Upsert: &upsert,
	})
}
