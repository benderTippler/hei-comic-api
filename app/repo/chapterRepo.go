package repo

import (
	"fmt"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/jinzhu/gorm"
	"go.mongodb.org/mongo-driver/bson"
	"hei-comic-api/app/collector/adapter"
	"hei-comic-api/app/httpio/in"
	"hei-comic-api/app/httpio/out"
	"hei-comic-api/app/middleware"
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

var (
	ChapterRepo     = new(chapterRepo)
	pageChapTerSize = 200
)

type chapterRepo struct{}

// GetComicChapters 漫画章节列表
func (r *chapterRepo) GetComicChapters(ctx middleware.MyCtx, in *in.GetComicChapters) (*out.ChapterOut, error) {
	var err error
	rsp := &out.ChapterOut{
		Chapters: make([]*out.Chapter, 0),
	}
	comic := &mongo.Comic{}
	mongodb := baseMongo.NewMongo().Database("comics").Collection("comics")
	// 查询mongodb 数据
	filter := make(bson.M, 0)
	filter["uuid"] = bson.M{
		"$eq": gconv.Uint64(in.ComicId),
	}
	result := mongodb.FindOne(ctx.Context, filter)
	if result.Err() != nil {
		return rsp, nil
	}
	err = result.Decode(comic)
	if err != nil {
		return rsp, err
	}
	//查看章节列表是否存在数据库
	db := mysql.NewDb()
	chapters := make([]*model.Chapter, 0)
	offset := (in.Page - 1) * pageChapTerSize
	err = db.Table((&model.Chapter{}).GetTableName(comic.OrderId)).Select("uuid,name,sort,pid,orderId,state").Where("pid = ?", comic.UUID).Limit(pageChapTerSize).Offset(offset).Order("sort asc").Find(&chapters).Error
	// 查询数据库章节列表
	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return rsp, nil
		}
		return rsp, err
	}

	var count int64
	db.Table((&model.Chapter{}).GetTableName(comic.OrderId)).Where("pid = ?", comic.UUID).Count(&count)

	for i, chapter := range chapters {
		rsp.Chapters = append(rsp.Chapters, &out.Chapter{
			UUID:    gconv.String(chapter.UUID),
			Name:    chapter.Name,
			Pid:     chapter.Pid,
			Sort:    offset + i + 1,
			OrderId: chapter.OrderId,
			State:   chapter.State,
		})
	}

	rsp.Count = gconv.Int(count)
	return rsp, nil
}

// GetComicChapterResource 漫画资源列表
func (r *chapterRepo) GetComicChapterResource(ctx middleware.MyCtx, req *in.ChapterResource) (*out.ChapterResourceOut, error) {
	var (
		resource []string
		err      error
	)
	chapterResourceOut := &out.ChapterResourceOut{
		Resource: make([]string, 0),
	}
	db := mysql.NewDb()
	chapter := &model.Chapter{}
	err = db.Table((&model.Chapter{}).GetTableName(gconv.Int(req.OrderId))).Where("uuid = ?", req.ChapterId).First(&chapter).Error
	if gorm.IsRecordNotFoundError(err) { //查询不到数据
		return chapterResourceOut, err //错误信息
	}

	var (
		imageCount int
		isReferer  bool
		referer    string
	)
	if chapter.State != 3 {
		adapter := adapter.NewAdapterCollector(chapter.OrderId)
		if chapter.Resources == "" {
			resource, err = adapter.GetResource(chapter.Target)
			if err != nil {
				fmt.Println("采集章节错误", err)
				return chapterResourceOut, err //错误信息
			}
			err = db.Table((&model.Chapter{}).GetTableName(gconv.Int(req.OrderId))).Where("uuid = ?", req.ChapterId).Update("resources", gstr.Join(resource, "|")).Error
		} else {
			resource = gstr.Split(chapter.Resources, "|")
		}
		isReferer, referer = adapter.IsReferer("resources")
	} else { //本地nas 资源服务
		resource = gstr.Split(chapter.DownloadPath, "|")
		for i, v := range resource {
			resource[i] = fmt.Sprintf("http://bender.tpddns.cn:5556/comics/%v", v)
		}
	}

	if isReferer {
		imageCount = 5
	} else {
		imageCount = 50
	}
	//分割成数组，4张一组
	pageSlice := utills.CutStringSlice(resource, imageCount)
	//这里要加个索引越界判断
	if req.Page > len(pageSlice) {
		return chapterResourceOut, nil
	}
	currentPageResource := pageSlice[req.Page-1]

	if isReferer { //开启防盗链接，这边要转换成base64图片格式返回
		wg := sync.WaitGroup{}
		for i, v := range currentPageResource {
			wg.Add(1)
			go func(i int, v string) {
				defer wg.Done()
				currentPageResource[i] = utills.ToImageBase64(v, referer)
			}(i, v)
			wg.Wait()
		}
		chapterResourceOut.ImageType = "base64"
	} else {
		chapterResourceOut.ImageType = "link"
	}
	if err != nil {
		return chapterResourceOut, err
	}
	chapterResourceOut.Resource = currentPageResource
	chapterResourceOut.Count = len(resource)
	chapterResourceOut.AllPage = len(pageSlice)
	return chapterResourceOut, nil
}

// 获取漫画前十章数据
func (r *chapterRepo) GetComicChapterTops(ctx middleware.MyCtx, in *in.GetComicChapterTops) (*out.ChapterOut, error) {
	var err error
	rsp := &out.ChapterOut{
		Chapters: make([]*out.Chapter, 0),
	}
	comic := &mongo.Comic{}
	mongodb := baseMongo.NewMongo().Database("comics").Collection("comics")
	// 查询mongodb 数据
	filter := make(bson.M, 0)
	filter["uuid"] = bson.M{
		"$eq": gconv.Uint64(in.ComicId),
	}
	result := mongodb.FindOne(ctx.Context, filter)
	if result.Err() != nil {
		return rsp, nil
	}
	err = result.Decode(comic)
	if err != nil {
		return rsp, err
	}
	//查看章节列表是否存在数据库
	db := mysql.NewDb()
	chapters := make([]*model.Chapter, 0)
	err = db.Table((&model.Chapter{}).GetTableName(comic.OrderId)).Where("pid = ?", comic.UUID).Limit(10).Order("sort asc").Find(&chapters).Error
	// 查询数据库章节列表
	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return rsp, nil
		}
		return rsp, err
	}
	adapter := adapter.NewAdapterCollector(comic.OrderId)
	isReferer, referer := adapter.IsReferer("resources")
	var count int64
	db.Table((&model.Chapter{}).GetTableName(comic.OrderId)).Where("pid = ?", comic.UUID).Count(&count)
	rand.Seed(time.Now().UnixNano())

	//开启协成
	wg := sync.WaitGroup{}
	for i, chapter := range chapters {
		wg.Add(1)
		go func(i int, chapter *model.Chapter) {
			defer wg.Done()
			var cover string
			var resource []string
			var prefix string
			var imageType string = "link"
			var num int
			if chapter.State == 3 {
				resource = gstr.Split(chapter.DownloadPath, "|")
				num = rand.Intn(len(resource))
				prefix = "http://bender.tpddns.cn:5556/comics/"
				cover = fmt.Sprintf("%v%v", prefix, resource[num])
			} else {
				resource = gstr.Split(chapter.Resources, "|")
				num = rand.Intn(len(resource))
				if isReferer {
					cover = utills.ToImageBase64(resource[num], referer)
					imageType = "base64"
				} else {
					cover = fmt.Sprintf("%v%v", prefix, resource[num])
				}
			}
			rsp.Chapters = append(rsp.Chapters, &out.Chapter{
				UUID:      gconv.String(chapter.UUID),
				Name:      chapter.Name,
				Pid:       chapter.Pid,
				Sort:      i + 1,
				OrderId:   chapter.OrderId,
				State:     chapter.State,
				Cover:     cover,
				ImageType: imageType,
			})
		}(i, chapter)
	}
	//排序一下
	sort.SliceStable(rsp.Chapters, func(i, j int) bool {
		return rsp.Chapters[i].Sort < rsp.Chapters[j].Sort
	})
	wg.Wait()
	rsp.Count = gconv.Int(count)
	return rsp, nil
}
