package bank

import (
	"fmt"
	"github.com/gocolly/colly"
	"github.com/gogf/gf/v2/text/gstr"
	"hei-comic-api/app/collector/adapter"
	"hei-comic-api/app/model"
	"time"
)

var MianZhuiAdapter = new(mianZhui)

type mianZhui struct {
	adapter.BaseAdapter
	scope []string
}

// 初始化 mianZhui 实例对象
func init() {
	err := MianZhuiAdapter.InitCfg(5) //初始化配置
	if err != nil {
		panic("MianZhuiAdapter InitCfg Fail")
	}
}

// StartUp 采集脚本入口
func (a *mianZhui) StartUp() error {
	return nil
}

// List 漫画列表页面 完结
func (a *mianZhui) List(target string) ([]*model.Comic, error) {
	comics := make([]*model.Comic, 0)
	cy := colly.NewCollector()
	cy.SetRequestTimeout(10 * time.Minute)
	cy.OnHTML(".cy_list_mh", func(e *colly.HTMLElement) {
		e.ForEach("ul", func(i int, item *colly.HTMLElement) {
			name := item.ChildText(".title")
			cover := item.ChildAttr(".pic > img", "src")
			detailUrl := fmt.Sprintf("%v%v", a.Origin, item.ChildAttr(".title > a", "href"))
			comics = append(comics, &model.Comic{
				Name:     name,
				Cover:    cover,
				Origin:   a.Origin,
				Target:   detailUrl,
				State:    0,
				OrderId:  a.OrderId,
				Language: a.Language,
			})
		})
	})
	err := cy.Visit(target)
	return comics, err
}

// Details 漫画详情
func (a *mianZhui) Details(comic *model.Comic) error {
	var err error
	url := comic.Target
	cy := colly.NewCollector()
	cy.SetRequestTimeout(10 * time.Minute)
	cy.OnHTML("body", func(e *colly.HTMLElement) {
		author := e.ChildText(".cy_author > a")
		stateStr := e.ChildText(".cy_serialize")
		catalogue := e.ChildText(".cy_type > a")
		if gstr.ContainsI(stateStr, "完结") {
			comic.State = 2
		}
		if gstr.ContainsI(stateStr, "连载") {
			comic.State = 1
		}
		comic.Author = author
		comic.Catalogue = gstr.Trim(catalogue)
		content := e.ChildText("#comic-description")
		comic.Content = gstr.Trim(gstr.ReplaceByMap(content, map[string]string{}))

	})
	err = cy.Visit(url)
	return err
}

// ChapterCount 获取章节总数 完毕
func (a *mianZhui) ChapterCount(comic *model.Comic) (int, error) {
	count := 0
	cy := colly.NewCollector()
	cy.OnHTML("#play_0", func(e *colly.HTMLElement) {
		count = e.DOM.Children().Find("li").Size()
	})
	err := cy.Visit(comic.Target)
	return count, err
}

// Chapters 获取章节数列表 完毕
func (a *mianZhui) Chapters(comic *model.Comic) ([]*model.Chapter, error) {
	chapters := make([]*model.Chapter, 0)
	cy := colly.NewCollector()
	cy.OnHTML("#mh-chapter-list-ol-0", func(e *colly.HTMLElement) {
		e.ForEach("li", func(i int, item *colly.HTMLElement) {
			name := item.ChildText("a > p")
			target := item.ChildAttr("a", "href")
			chapters = append(chapters, &model.Chapter{
				Name:    gstr.Trim(name),
				Pid:     comic.UUID,
				State:   0,
				Sort:    i,
				Target:  fmt.Sprintf("%v%v", a.Origin, target),
				Origin:  a.Origin,
				OrderId: a.OrderId,
			})
		})
	})
	err := cy.Visit(comic.Target)
	return chapters, err
}

// GetResource 漫画资源 完毕
func (a *mianZhui) GetResource(targetUrl string) ([]string, error) {
	resource := make([]string, 0)
	cy := colly.NewCollector()
	cy.SetRequestTimeout(10 * time.Minute)
	cy.OnHTML(".rd-article-wr", func(e *colly.HTMLElement) {
		e.ForEach(".rd-article__pic", func(i int, item *colly.HTMLElement) {
			cover := item.ChildAttr("img", "data-original")
			resource = append(resource, cover)
		})
	})
	err := cy.Visit(targetUrl)
	return resource, err
}
