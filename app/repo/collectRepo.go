package repo

import (
	"errors"
	"github.com/gogf/gf/v2/util/gconv"
	"go.mongodb.org/mongo-driver/bson"
	"gorm.io/gorm"
	baseError "hei-comic-api/app/error"
	"hei-comic-api/app/httpio/in"
	"hei-comic-api/app/httpio/out"
	"hei-comic-api/app/middleware"
	"hei-comic-api/app/model"
	"hei-comic-api/app/model/mongo"
	baseMongo "hei-comic-api/base/mongo"
	"hei-comic-api/base/mysql"
)

var (
	CollectRepo = new(collectRepo)
)

type collectRepo struct{}

func (r collectRepo) CreateCollect(ctx middleware.MyCtx, userId int64, req *in.CollectIn) error {
	db := mysql.NewDb()
	comicId := gconv.Int64(req.ComicId)
	chapterId := gconv.Int64(req.ChapterId)
	var chapterM = &model.Chapter{}
	collectM := &model.Collect{}
	err := db.Model(&model.Collect{}).Where("comicId = ? and uid = ?", comicId, userId).First(&collectM).Error

	if req.IsCollect == 1 {
		if collectM.UID > 0 {
			return baseError.UserExistCollectErr
		}
		if chapterId == 0 { //默认收藏第一个
			err = db.Table((&model.Chapter{}).GetTableName(req.OrderId)).Where("pid = ?", comicId).Order("sort asc").Limit(1).First(&chapterM).Error
			if err != nil {
				return baseError.UserCreateCollectErr
			}
		}
		collect := &model.Collect{
			UID:       userId,
			ComicId:   comicId,
			ChapterId: chapterM.UUID,
		}
		err = db.Model(&model.Collect{}).Create(collect).Error
		if err != nil {
			return baseError.UserCreateCollectErr
		}
	} else if req.IsCollect == 2 { //取消收藏
		err = db.Where("comicId = ? and uid = ?", comicId, userId).Delete(&model.Collect{}).Error
		if err != nil {
			return baseError.UserQXCollectErr
		}
	}
	return nil
}

func (r collectRepo) UserIsCollectComicId(ctx middleware.MyCtx, userId int64, req *in.UserIsCollectComicIdIn) (*out.UserIsCollectComicId, error) {
	db := mysql.NewDb()
	comicId := gconv.Int64(req.ComicId)
	out := &out.UserIsCollectComicId{}
	collectM := &model.Collect{}
	err := db.Model(&model.Collect{}).Where("comicId = ? and uid = ?", comicId, userId).First(&collectM).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			out.CollectType = 1
			return out, nil
		}
		return out, baseError.SystemMysqlErr

	}
	out.CollectType = 2
	return out, nil
}

// UserCollects 用户收藏漫画列表
func (r collectRepo) UserCollects(ctx middleware.MyCtx, userId int64) (*out.UserCollectsOut, error) {
	db := mysql.NewDb()
	out := &out.UserCollectsOut{}
	comicIds := make([]int64, 0)
	err := db.Model(&model.Collect{}).Where("uid = ?", userId).Select("comicId").Order("updateTime desc").Scan(&comicIds).Error
	if err != nil {
		return out, err
	}
	if len(comicIds) > 0 {
		mongodb := baseMongo.NewMongo().Database("comics").Collection("comics")
		//查询条件
		filter := make(bson.M, 0)
		filter["uuid"] = bson.M{
			"$in": comicIds,
		}
		cur, err := mongodb.Find(ctx.Context, filter)
		if err != nil {
			return out, err
		}
		comics := make([]*mongo.Comic, 0, 21)
		defer cur.Close(ctx.Context)
		err = cur.All(ctx.Context, &comics)
		out.Comics = comics
	}
	return out, nil
}
