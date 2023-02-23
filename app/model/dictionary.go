package model

import (
	"gorm.io/gorm"
	"hei-comic-api/base/snowflake"
)

type Dictionary struct {
	UUID    int64  `gorm:"column:uuid" db:"uuid" json:"uuid" form:"uuid"`
	Title   string `gorm:"column:title" db:"title" json:"title" form:"title"`
	Icon    string `gorm:"column:icon" db:"icon" json:"icon" form:"icon"`
	Field   string `gorm:"column:field" db:"field" json:"field" form:"field"`
	Sort    int    `gorm:"column:sort" db:"sort" json:"sort" form:"sort"`
	Content string `gorm:"column:content" db:"content" json:"content" form:"content"`
}

// TableName 数据库表表明
func (c *Dictionary) TableName() string {
	return "dictionary"
}

func (c *Dictionary) BeforeCreate(db *gorm.DB) error {
	node := snowflake.NewSnowNode()
	c.UUID = node.Generate().Int64()
	return nil
}
