package model

import (
	"gorm.io/gorm"
	"hei-comic-api/base/snowflake"
	"time"
)

type Collect struct {
	UUID       int64 `gorm:"column:uuid" json:"uuid"`
	UID        int64 `gorm:"column:uid" json:"uid"`
	ComicId    int64 `gorm:"column:comicId" json:"comicId"`
	ChapterId  int64 `gorm:"column:chapterId" json:"chapterId"`
	CreateTime int64 `gorm:"column:createTime" json:"createTime"`
	UpdateTime int64 `gorm:"column:updateTime" json:"updateTime"`
}

// TableName 数据库表表明
func (c *Collect) TableName() string {
	return "collect"
}

func (c *Collect) BeforeCreate(db *gorm.DB) error {
	node := snowflake.NewSnowNode()
	time := time.Now()
	c.UUID = node.Generate().Int64()
	c.CreateTime = time.Unix()
	c.UpdateTime = time.Unix()
	return nil
}

func (c *Collect) BeforeUpdate(db *gorm.DB) error {
	time := time.Now()
	c.UpdateTime = time.Unix()
	return nil
}
