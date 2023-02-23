package out

import (
	"github.com/gogf/gf/v2/util/gconv"
	"hei-comic-api/app/model"
	"hei-comic-api/app/model/mongo"
)

type Comic struct {
	UUID      string   `json:"uuid"`      //雪花id
	Name      string   `json:"name"`      // 漫画名称
	State     int      `json:"state"`     // 1、连载  2、完结
	Catalogue []string `json:"catalogue"` // 分类
	Cover     string   `json:"cover"`     // 封面地址
	Language  string   `json:"language"`  //漫画语言
	OrderId   int      `json:"orderId"`   //适配编码
	Origin    string   `json:"origin"`
	Content   string   `json:"content"`
}

func (c *Comic) Unmarshals(in []*mongo.Comic) []*Comic {
	out := make([]*Comic, 0)
	for _, comic := range in {
		out = append(out, &Comic{
			UUID:      gconv.String(comic.UUID),
			Name:      comic.Name,
			State:     comic.State,
			Catalogue: comic.Catalogue,
			Cover:     comic.Cover,
			Language:  comic.Language,
			OrderId:   comic.OrderId,
			Origin:    comic.Origin,
			Content:   comic.Content,
		})
	}
	return out
}

type ComicSetting struct {
	State    interface{} `json:"state"`
	Language interface{} `json:"language"`
	Classify interface{} `json:"classify"`
}

type ComicList struct {
	UUID       string `json:"uuid"`    //雪花id
	Name       string `json:"name"`    // 漫画名称
	Cover      string `json:"cover"`   // 封面地址
	OrderId    int    `json:"orderId"` //适配编码
	State      int    `json:"state"`   // 1、连载  2、完结
	Origin     string `json:"origin"`
	Content    string `json:"content"`
	CoverLocal string `json:"coverLocal"`
}

func (c *ComicList) Unmarshals(in []*mongo.Comic) []*ComicList {
	out := make([]*ComicList, 0)
	for _, comic := range in {
		out = append(out, &ComicList{
			UUID:    gconv.String(comic.UUID),
			Name:    comic.Name,
			Cover:   comic.Cover,
			OrderId: comic.OrderId,
			State:   comic.State,
			Origin:  comic.Origin,
			Content: comic.Content,
		})
	}
	return out
}

type ComparComicName struct {
	Name string `bson:"_id"`
}

// ComparisonComic 后台手动比较数据
type ComparisonComic struct {
	UUID    string             `json:"uuid"`    //雪花id
	Name    string             `json:"name"`    // 漫画名称
	Cover   string             `json:"cover"`   // 封面地址
	OrderId int                `json:"orderId"` //适配编码
	State   int                `json:"state"`   // 1、连载  2、完结
	Origin  string             `json:"origin"`  //资源地址
	Content string             `json:"content"` //资源内容
	Author  string             `json:"author"`  //作者
	Chapter *ComparisonChapter `json:"chapter"` //章节数据
}

func (c *ComparisonComic) Unmarshals(in []*mongo.Comic) []*ComparisonComic {
	out := make([]*ComparisonComic, 0)
	for _, comic := range in {
		out = append(out, &ComparisonComic{
			UUID:    gconv.String(comic.UUID),
			Name:    comic.Name,
			Cover:   comic.Cover,
			OrderId: comic.OrderId,
			State:   comic.State,
			Origin:  comic.Origin,
			Content: comic.Content,
			Author:  comic.Author,
		})
	}
	return out
}

type ComparisonChapter struct {
	Total          int              `json:"total"`
	TotalChapters  []*model.Chapter `json:"totalChapters"`
	Normal         int              `json:"normal"`
	NormalChapters []*model.Chapter `json:"normalChapters"`
	Damage         int              `json:"damage"`
	DamageChapters []*model.Chapter `json:"damageChapters"`
}
