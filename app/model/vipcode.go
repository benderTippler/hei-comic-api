package model

import (
	"gorm.io/gorm"
	"hei-comic-api/base/snowflake"
	"time"
)

type VipCode struct {
	UUID       int64  `gorm:"column:uuid" db:"uuid" json:"uuid"`
	State      int    `gorm:"column:state" db:"state" json:"state"` //0 未被使用 1 使用
	VipCode    string `gorm:"column:vipCode" db:"vipCode" json:"vipCode"`
	Target     int64  `gorm:"column:target" db:"target" json:"target"`
	OpenId     string `gorm:"column:openId" db:"openId" json:"openId"`
	CreateTime int64  `gorm:"column:createTime" db:"createTime" json:"createTime"`
	UpdateTime int64  `gorm:"column:updateTime" db:"updateTime" json:"updateTime"`
}

func (c *VipCode) TableName() string {
	return "vipCodes"
}

func (c *VipCode) BeforeCreate(db *gorm.DB) error {
	node := snowflake.NewSnowNode()
	time := time.Now()
	c.UUID = node.Generate().Int64()
	c.CreateTime = time.Unix()
	c.UpdateTime = time.Unix()
	return nil
}

func (c *VipCode) BeforeUpdate(db *gorm.DB) error {
	time := time.Now()
	c.UpdateTime = time.Unix()
	return nil
}
