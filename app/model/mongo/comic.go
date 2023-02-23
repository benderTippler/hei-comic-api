package mongo

import (
	"github.com/gogf/gf/v2/text/gstr"
	"hei-comic-api/app/model"
)

type Comic struct {
	UUID       int64    `bson:"uuid" json:"uuid"`             //雪花id
	Name       string   `bson:"name" json:"name"`             // 漫画名称
	Author     string   `bson:"author" json:"author"`         // 作者名称
	State      int      `bson:"state" json:"state"`           // 1、连载  2、完结
	Catalogue  []string `bson:"catalogue" json:"catalogue"`   // 分类
	Cover      string   `bson:"cover" json:"cover"`           // 封面地址
	Content    string   `bson:"content" json:"content"`       // 内容
	Origin     string   `bson:"origin" json:"origin"`         // 资源来源
	Target     string   `bson:"target" json:"target"`         // 漫画详情地址
	Language   string   `bson:"language" json:"language"`     //漫画语言
	OrderId    int      `bson:"orderId" json:"orderId"`       //适配编码
	Status     int      `bson:"status" json:"status"`         // 状态 0 上架  1、下架
	IsHandle   int      `bson:"isHandle" json:"isHandle"`     // 采集信息是否补齐 只针对完结动漫而言
	CreateTime int64    `bson:"createTime" json:"createTime"` //创建时间
	UpdateTime int64    `bson:"updateTime" json:"updateTime"` //更新时间
}

// model 转换 mongod数据
func (m *Comic) Unmarshal(c *model.Comic) {
	m.UUID = c.UUID
	m.Name = c.Name
	m.Author = c.Author
	m.State = c.State
	catalogue := make([]string, 0)
	if c.Catalogue == "" {
		if c.Language == "zh" {
			catalogue = append(catalogue, "其他")
		} else {
			catalogue = append(catalogue, "other")
		}
	} else {
		for _, v := range gstr.Split(c.Catalogue, ",") {
			if v != "" && len(v) <= 6 {
				catalogue = append(catalogue, gstr.Trim(v))
			}
		}
	}
	m.Catalogue = catalogue
	m.Cover = c.Cover
	m.Content = c.Content
	m.Origin = c.Origin
	m.Target = c.Target
	m.Language = c.Language
	m.OrderId = c.OrderId
	m.Status = c.Status
	m.IsHandle = c.IsHandle
	m.CreateTime = c.CreateTime
	m.UpdateTime = c.UpdateTime
}

// model数据 转换 mong
func (m *Comic) MarshalToModelComic() *model.Comic {
	return &model.Comic{
		UUID:       m.UUID,
		Name:       m.Name,
		Author:     m.Author,
		State:      m.State,
		Catalogue:  gstr.Join(m.Catalogue, " "),
		Cover:      m.Cover,
		Content:    m.Content,
		Origin:     m.Origin,
		Target:     m.Target,
		Language:   m.Language,
		OrderId:    m.OrderId,
		Status:     m.Status,
		IsHandle:   m.IsHandle,
		CreateTime: m.CreateTime,
		UpdateTime: m.UpdateTime,
	}
}
