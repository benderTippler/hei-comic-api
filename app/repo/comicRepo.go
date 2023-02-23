package repo

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/jinzhu/gorm"
	"go.mongodb.org/mongo-driver/bson"
	mongo2 "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"hei-comic-api/app/collector/adapter"
	"hei-comic-api/app/httpio/in"
	"hei-comic-api/app/httpio/out"
	"hei-comic-api/app/middleware"
	"hei-comic-api/app/model"
	"hei-comic-api/app/model/mongo"
	"hei-comic-api/app/utills"
	baseMongo "hei-comic-api/base/mongo"
	"hei-comic-api/base/mysql"
	"hei-comic-api/base/redis"
	"sync"
	"time"
)

var (
	ComicRepo     = new(comicRepo)
	limit     int = 30
)

type comicRepo struct{}

// FindComicList  漫画列表
func (r *comicRepo) FindComicList(ctx middleware.MyCtx, req *in.ComicList) ([]*out.ComicList, error) {
	var err error
	rsp := make([]*out.ComicList, 0)
	// 1、mongodb 分页
	mongodb := baseMongo.NewMongo().Database("comics").Collection("comics")
	var findOpts = &options.FindOptions{}

	// 分页配置
	offset := (req.Page - 1) * limit
	findOpts.SetLimit(gconv.Int64(limit))
	findOpts.SetSkip(gconv.Int64(offset))
	sort := bson.D{{"updateTime", -1}}
	findOpts.SetSort(sort)

	//查询条件
	filter := make(bson.M, 0)

	if req.State > 0 {
		filter["state"] = req.State
	}
	//filter["status"] = 88
	filter["status"] = bson.M{
		"$in": []int{
			2, 88,
		},
	}
	// 开启来源验证
	//filter["orderId"] = bson.M{
	//	"$in": []int{
	//		8,
	//	},
	//}

	if req.Name != "" {
		filter["name"] = bson.M{
			"$regex":   req.Name,
			"$options": "-i",
		}
	}

	if req.Language != "" {
		filter["language"] = req.Language
	}

	if req.Catalogue != "" {
		filter["catalogue"] = bson.M{
			"$in": []string{
				req.Catalogue,
			},
		}
	}

	cur, err := mongodb.Find(ctx.Context, filter, findOpts)
	if err != nil {
		return rsp, nil
	}
	comics := make([]*mongo.Comic, 0, 21)
	defer cur.Close(ctx.Context)
	err = cur.All(ctx.Context, &comics)
	rsp = (&out.ComicList{}).Unmarshals(comics)

	wg := sync.WaitGroup{}
	for _, comic := range rsp {
		wg.Add(1)
		go func(comic *out.ComicList) {
			defer wg.Done()
			if comic.CoverLocal == "" {
				isReferer, referer := adapter.NewAdapterCollector(comic.OrderId).IsReferer("cover")
				if isReferer { //开启防盗链接，这边要转换成base64图片格式返回
					comic.Cover = utills.ToImageBase64(comic.Cover, referer)
				}
			} else {
				comic.Cover = fmt.Sprintf("http://bender.tpddns.cn:5556/comics/%v", comic.CoverLocal)
			}
		}(comic)
	}
	wg.Wait()

	if err != nil {
		return rsp, err
	}
	return rsp, nil
}

// GetComicById 漫画详情 通过Id
func (r *comicRepo) GetComicById(ctx middleware.MyCtx, uuid uint64) (*mongo.Comic, error) {
	var err error
	comic := &mongo.Comic{}
	mongodb := baseMongo.NewMongo().Database("comics").Collection("comics")
	if uuid <= 0 {
		return comic, errors.New("参数错误")
	}
	filter := make(bson.M, 0)
	filter["uuid"] = bson.M{
		"$eq": uuid,
	}
	result := mongodb.FindOne(ctx.Context, filter)
	if result.Err() != nil {
		return comic, nil
	}
	err = result.Decode(comic)
	if err != nil {
		return comic, err
	}
	return comic, nil
}

// GetComicByTarget 获取是否存在
func (r *comicRepo) GetComicByTarget(ctx middleware.MyCtx, target string) (*model.Comic, error) {
	var err error
	comic := &model.Comic{}
	if target == "" {
		return comic, errors.New("参数错误")
	}
	db := mysql.NewDb()
	err = db.Table("comics").Where("target = ?", target).First(comic).Error
	if gorm.IsRecordNotFoundError(err) {
		return comic, nil
	}
	return comic, nil
}

func (r *comicRepo) GetComicSetting(ctx middleware.MyCtx) (*out.ComicSetting, error) {
	redis := redis.NewRedis()
	setting := &out.ComicSetting{}
	redisKey := "setting"
	result, err := redis.Get(ctx.Context, redisKey).Result()
	if err == nil && result != "" {
		err = json.Unmarshal([]byte(result), &setting)
		if err == nil {
			return setting, nil
		}
	}
	setting.State = map[string]interface{}{
		"zh": map[int]string{
			1: "连载中",
			2: "已完成",
		},
		"en": map[int]string{
			1: "ongoing",
			2: "completed",
		},
	}

	setting.Language = map[string]interface{}{
		"zh": map[int]string{
			1: "中文",
			2: "英文",
		},
		"en": map[int]string{
			1: "chinese",
			2: "english",
		},
	}

	setting.Classify = map[string]interface{}{
		"zh": r.getClassify(ctx, "zh"),
		"en": r.getClassify(ctx, "en"),
	}

	redis.Set(ctx.Context, redisKey, gconv.String(setting), 0)

	return setting, nil
}

func (r *comicRepo) getClassify(ctx middleware.MyCtx, search string) []string {
	classify := make([]string, 0)
	classifyChMap := make(map[string]string)
	mongodb := baseMongo.NewMongo().Database("comics").Collection("comics")
	cur, err := mongodb.Find(ctx.Context, bson.M{
		"language": search,
	})
	if err != nil {
		return classify
	}
	comics := make([]*mongo.Comic, 0)
	defer cur.Close(ctx.Context)
	err = cur.All(ctx.Context, &comics)
	for _, comic := range comics {
		for _, v := range comic.Catalogue {
			if len(v) <= 6 {
				if _, ok := classifyChMap[v]; !ok {
					classifyChMap[v] = v
					classify = append(classify, v)
				}
			}
		}
	}
	return classify
}

// ComparisonComic 手动比较数据
func (r *comicRepo) ComparisonComic(ctx middleware.MyCtx, req *in.ComparisonComic) (map[string][]*out.ComparisonComic, error) {
	var err error
	rsp := make(map[string][]*out.ComparisonComic, 0)
	// 1、mongodb 分页
	mongodb := baseMongo.NewMongo().Database("comics").Collection("comics")
	//开始  分组查询数据
	comicNames := make([]*out.ComparComicName, 0)
	groupFilter := make(bson.M, 0)
	if req.State > 0 {
		groupFilter["state"] = req.State
	}
	if req.Name != "" {
		groupFilter["name"] = bson.M{
			"$regex":   req.Name,
			"$options": "-i",
		}
	}
	offset := (req.Page - 1) * limit
	pipeline := mongo2.Pipeline{
		bson.D{ //排序
			{"$sort", bson.D{{"name", -1}}},
		},
		bson.D{ //查询条件
			{"$match", groupFilter},
		},
		bson.D{
			{"$limit", limit},
		},
		bson.D{
			{"$skip", offset},
		},
		bson.D{
			{
				"$group", bson.D{
					{"_id", "$name"},
				},
			},
		},
	}
	opts := options.Aggregate()
	cursor, err := mongodb.Aggregate(ctx.Context, pipeline, opts)
	defer cursor.Close(ctx.Context)
	if err = cursor.All(ctx.Context, &comicNames); err != nil {
		return rsp, nil
	}
	//结束  分组查询数据
	for _, comicName := range comicNames {
		// 分页配置
		comicRsp := make([]*out.ComparisonComic, 0)
		var findOpts = &options.FindOptions{}
		sort := bson.D{{"name", -1}, {"updateTime", -1}}
		findOpts.SetSort(sort)
		//查询条件
		filter := make(bson.M, 0)
		filter["name"] = comicName.Name
		cur, err := mongodb.Find(ctx.Context, filter, findOpts)
		if err != nil {
			continue
		}
		comics := make([]*mongo.Comic, 0, 21)
		defer cur.Close(ctx.Context)
		err = cur.All(ctx.Context, &comics)
		if err != nil {
			continue
		}
		comicRsp = (&out.ComparisonComic{}).Unmarshals(comics)
		wg := sync.WaitGroup{}
		db := mysql.NewDb()
		for _, comic := range comicRsp {
			wg.Add(1)
			go func(comic *out.ComparisonComic) {
				defer wg.Done()
				isReferer, referer := adapter.NewAdapterCollector(comic.OrderId).IsReferer("cover")
				if isReferer { //开启防盗链接，这边要转换成base64图片格式返回
					comic.Cover = utills.ToImageBase64(comic.Cover, referer)
				}
				var (
					TotalChapters  = make([]*model.Chapter, 0)
					NormalChapters = make([]*model.Chapter, 0)
					DamageChapters = make([]*model.Chapter, 0)
				)
				db.Table((&model.Chapter{}).GetTableName(comic.OrderId)).Select("uuid,name,state").Where("pid = ?", comic.UUID).Find(&TotalChapters)
				for _, v := range TotalChapters {
					if v.State == 1 {
						NormalChapters = append(NormalChapters, v)
					} else if v.State == 2 {
						DamageChapters = append(DamageChapters, v)
					}
				}
				ComparisonChapter := &out.ComparisonChapter{
					Total:          len(TotalChapters),
					TotalChapters:  TotalChapters,
					Normal:         len(NormalChapters),
					NormalChapters: NormalChapters,
					Damage:         len(DamageChapters),
					DamageChapters: DamageChapters,
				}
				comic.Chapter = ComparisonChapter
			}(comic)
		}
		wg.Wait()
		rsp[comicName.Name] = comicRsp
	}
	if err != nil {
		return rsp, err
	}
	return rsp, nil
}

func (r *comicRepo) ComicUpdate(ctx middleware.MyCtx) (*out.ComicUpt, error) {
	comicUpt := &out.ComicUpt{
		Source: 7,
	}
	//查询漫画更新数据
	mongodb := baseMongo.NewMongo().Database("comics").Collection("comics")
	filter := make(bson.M, 0)
	filter["status"] = bson.M{
		"$in": []int{
			2, 88,
		},
	}
	count, _ := mongodb.CountDocuments(ctx.Context, filter)
	comicUpt.Count = count
	now := time.Now()
	startTime := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	filter["updateTime"] = bson.M{
		"$gt": startTime.Unix(),
		"$lt": now.Unix(),
	}
	dayUptCount, _ := mongodb.CountDocuments(ctx.Context, filter)
	comicUpt.DayUpdateCount = dayUptCount
	comicUpt.Version = "v1.0.0"
	return comicUpt, nil
}
