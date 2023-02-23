package model

import (
	"gorm.io/gorm"
	"hei-comic-api/base/snowflake"
	"time"
)

type User struct {
	UUID          int64  `gorm:"column:uuid" json:"uuid"`
	Email         string `gorm:"column:email" json:"email"`                 //邮箱
	State         int    `gorm:"column:state" json:"state"`                 //0 启用  1 禁用 2、取消关注
	Phone         string `gorm:"column:phone" json:"phone"`                 //手机号
	Sex           int    `gorm:"column:sex" json:"sex"`                     //0 未知 1 男 2 女
	Device        string `gorm:"column:device" json:"device"`               //设备号
	CreateTime    int64  `gorm:"column:createTime" json:"createTime"`       //注册时间
	UpdateTime    int64  `gorm:"column:updateTime" json:"updateTime"`       //修改信息时间
	LastLoginTime int64  `gorm:"column:lastLoginTime" json:"lastLoginTime"` //最近登录时间
	VipCode       string `gorm:"column:vipCode" json:"vipCode"`             //邀请码
	Salt          string `gorm:"column:salt" json:"salt"`                   //干扰码
	Password      string `gorm:"column:password" json:"password"`           //加密密码
}

// TableName 数据库表表明
func (c *User) TableName() string {
	return "users"
}

func (c *User) BeforeCreate(db *gorm.DB) error {
	node := snowflake.NewSnowNode()
	time := time.Now()
	c.UUID = node.Generate().Int64()
	c.CreateTime = time.Unix()
	c.UpdateTime = time.Unix()
	return nil
}

func (c *User) BeforeUpdate(db *gorm.DB) error {
	time := time.Now()
	c.UpdateTime = time.Unix()

	return nil
}
