package repo

import (
	"hei-comic-api/app/httpio/admin/in"
	"hei-comic-api/app/httpio/admin/out"
	"hei-comic-api/app/middleware"
	"hei-comic-api/app/model"
	"hei-comic-api/base/mysql"
)

var (
	DictionaryRepo = new(dictionaryRepo)
)

type dictionaryRepo struct{}

// CreateDict 创建字典数据
func (r *dictionaryRepo) CreateDict(ctx middleware.MyCtx, dic *in.CreateDictIn) error {
	db := mysql.NewDb()
	dictM := &model.Dictionary{
		Field:   dic.Field,
		Content: dic.Content,
	}
	err := db.Model(&model.Dictionary{}).Create(&dictM).Error
	if err != nil {
		return err
	}
	return nil
}

// 获取字典数据
func (r *dictionaryRepo) FindDict(ctx middleware.MyCtx) (*out.FindDictOut, error) {
	db := mysql.NewDb()
	out := &out.FindDictOut{}
	dictMs := make([]*model.Dictionary, 0)
	err := db.Model(&model.Dictionary{}).Find(&dictMs).Error
	if err != nil {
		return out, err
	}
	out.Dictionary = dictMs
	return out, nil
}
