package model

import (
	"fmt"
	"gorm.io/gorm"
	"hei-comic-api/base/snowflake"
	"time"
)

// Comic 漫画表
/**
88、上架
1、下架
2、可用(收集到的漫画是数据库中唯一的，章节完整度比较高)
3、备用数据（存在缺少或者数据重复）
4、未找到最优漫画资源
5、针对包子漫画网，资源采集比较慢，用来占位
6、资源封版，(针对完结漫画数据，下载到本地，章节全部完整)
*/
type Comic struct {
	UUID       int64  `gorm:"column:uuid;default:0;NOT NULL" json:"uuid" bson:"uuid"`                   //雪花id
	Name       string `gorm:"column:name" json:"name" bson:"name"`                                      // 漫画名称
	Author     string `gorm:"column:author" json:"author" bson:"author"`                                // 作者名称
	State      int    `gorm:"column:state;default:0;NOT NULL" json:"state" bson:"state"`                // 1、连载  2、完结
	Catalogue  string `gorm:"column:catalogue" json:"catalogue" bson:"catalogue"`                       // 分类
	Cover      string `gorm:"column:cover" json:"cover" bson:"cover"`                                   // 封面地址
	Content    string `gorm:"column:content" json:"content" bson:"content"`                             // 内容
	Origin     string `gorm:"column:origin" json:"origin" bson:"origin"`                                // 资源来源
	Target     string `gorm:"column:target" json:"target" bson:"target"`                                // 漫画详情地址
	Language   string `gorm:"column:language" json:"language" bson:"language"`                          //漫画语言
	OrderId    int    `gorm:"column:orderId;default:0;NOT NULL" json:"orderId" bson:"orderId"`          //适配编码
	Status     int    `gorm:"column:status;default:0;NOT NULL" json:"status" bson:"status"`             // 状态 0 默认初始化状态
	IsHandle   int    `gorm:"column:isHandle;default:0;NOT NULL" json:"isHandle" bson:"isHandle"`       // 采集信息是否补齐 44、章节数据存在损坏
	CreateTime int64  `gorm:"column:createTime;default:0;NOT NULL" json:"createTime" bson:"createTime"` //创建时间
	UpdateTime int64  `gorm:"column:updateTime;default:0;NOT NULL" json:"updateTime" bson:"updateTime"` //更新时间
	CoverLocal string `gorm:"column:coverLocal" json:"coverLocal" bson:"coverLocal"`
}

// TableName 数据库表表明
func (c *Comic) TableName() string {
	return "comics"
}

func (c *Comic) BeforeCreate(db *gorm.DB) error {
	node := snowflake.NewSnowNode()
	time := time.Now()
	c.UUID = node.Generate().Int64()
	c.CreateTime = time.Unix()
	c.UpdateTime = time.Unix()
	return nil
}

func (c *Comic) BeforeUpdate(db *gorm.DB) error {
	time := time.Now()
	c.UpdateTime = time.Unix()
	return nil
}

func (c *Comic) GetTableName(orderId int) string {
	return fmt.Sprintf("%v_%v", c.TableName(), orderId)
}

// GetCreateTable 创建数据库表语句
func (c *Comic) GetCreateTable(orderId int, comment string) string {
	var create = `
CREATE TABLE comics_%v (
  uuid bigint(25) unsigned DEFAULT 0,
  name varchar(255) COLLATE utf8mb4_bin DEFAULT '',
  author varchar(255) COLLATE utf8mb4_bin DEFAULT '',
  state tinyint(3) unsigned DEFAULT 0,
  catalogue varchar(255) COLLATE utf8mb4_bin DEFAULT '',
  cover varchar(255) COLLATE utf8mb4_bin DEFAULT '',
  coverLocal varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT '',
  content longtext COLLATE utf8mb4_bin DEFAULT NULL,
  origin varchar(255) COLLATE utf8mb4_bin DEFAULT '',
  target varchar(255) COLLATE utf8mb4_bin DEFAULT '',
  language varchar(30) COLLATE utf8mb4_bin DEFAULT '',
  orderId int(10) unsigned DEFAULT 0,
  status tinyint(3) unsigned DEFAULT 0,
  isHandle tinyint(3) unsigned DEFAULT 0,
  createTime int(12) unsigned DEFAULT 0,
  updateTime int(12) unsigned DEFAULT 0,
  UNIQUE KEY target (target) USING BTREE,
  UNIQUE KEY uuid (uuid) USING BTREE,
  KEY origin (origin) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT="%v,漫画表";
`
	return fmt.Sprintf(create, orderId, comment)
}
