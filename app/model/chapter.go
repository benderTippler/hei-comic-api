package model

import (
	"fmt"
	"gorm.io/gorm"
	"hei-comic-api/base/snowflake"
	"time"
)

type Chapter struct {
	UUID         int64  `gorm:"column:uuid;default:0;NOT NULL" json:"id"`               //雪花id
	Name         string `gorm:"column:name" json:"name"`                                // 章节名称
	Resources    string `gorm:"column:resources" json:"resources"`                      // 资源图片，用|分割
	Pid          int64  `gorm:"column:pid" json:"pid"`                                  // 主表漫画表
	State        int    `gorm:"column:state;default:0" json:"state"`                    //  1、初始化网站数据成功，2、初始化时数据失败（部分图片缺失） 3、资源全部本地化，并且资源没有问题
	Sort         int    `gorm:"column:sort;default:0" json:"sort"`                      // 章节排序
	Target       string `gorm:"column:target" json:"target"`                            // 漫画详情地址
	Origin       string `gorm:"column:origin" json:"origin"`                            // 资源来源
	OrderId      int    `gorm:"column:orderId;default:0;NOT NULL" json:"orderId"`       //适配编码
	CreateTime   int64  `gorm:"column:createTime;default:0;NOT NULL" json:"createTime"` //创建时间
	UpdateTime   int64  `gorm:"column:updateTime;default:0;NOT NULL" json:"updateTime"` //更新时间
	DownloadPath string `gorm:"column:downloadPath" json:"downloadPath"`
}

// TableName 数据库表表明
func (c *Chapter) TableName() string {
	return "chapters"
}

func (c *Chapter) BeforeCreate(db *gorm.DB) error {
	node := snowflake.NewSnowNode()
	time := time.Now()
	c.UUID = node.Generate().Int64()
	c.CreateTime = time.Unix()
	c.UpdateTime = time.Unix()
	return nil
}

func (c *Chapter) BeforeUpdate(db *gorm.DB) error {
	time := time.Now()
	c.UpdateTime = time.Unix()
	return nil
}

func (c *Chapter) GetTableName(orderId int) string {
	return fmt.Sprintf("%v_%v", c.TableName(), orderId)
}

// GetCreateTable 创建数据库表语句
func (c *Chapter) GetCreateTable(orderId int, comment string) string {
	var create = `
CREATE TABLE chapters_%v (
  uuid bigint(25) unsigned DEFAULT 0,  
  name varchar(255) COLLATE utf8mb4_bin DEFAULT '',
  resources longtext COLLATE utf8mb4_bin DEFAULT NULL,
  pid bigint(25) unsigned DEFAULT 0,
  state tinyint(3) unsigned DEFAULT 0,
  sort int(10) unsigned DEFAULT 0,
  target varchar(255) COLLATE utf8mb4_bin DEFAULT '',
  origin varchar(255) COLLATE utf8mb4_bin DEFAULT '',
  orderId int(10) unsigned DEFAULT NULL,
  createTime int(12) unsigned DEFAULT 0,
  updateTime int(12) unsigned DEFAULT 0,
  downloadPath longtext COLLATE utf8mb4_bin,
  UNIQUE KEY target (target) USING BTREE,
  UNIQUE KEY uuid (uuid) USING BTREE,  
  KEY pid (pid) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin  COMMENT="%v,章节表";
`
	return fmt.Sprintf(create, orderId, comment)
}
