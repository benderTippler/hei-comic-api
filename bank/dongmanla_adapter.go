package bank

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
	"github.com/gocolly/colly/extensions"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
	"hei-comic-api/app/collector/adapter"
	"hei-comic-api/app/model"
	"regexp"
	"time"
)

var DongManLaAdapter = new(dongManLa)

type dongManLa struct {
	adapter.BaseAdapter
}

// 初始动漫啦实例对象
func init() {
	err := DongManLaAdapter.InitCfg(3) //初始化配置
	if err != nil {
		panic("dongManLa InitCfg Fail")
	}
}

// StartUp 采集脚本入口
func (a *dongManLa) StartUp() error {
	return nil
}

func (a *dongManLa) GetUpdateComics() ([]*model.Comic, error) {
	//comicSet := gset.New(true)
	comics := make([]*model.Comic, 0)
	return comics, nil
}

// List 漫画列表页面 完毕
func (a *dongManLa) List(target string) ([]*model.Comic, error) {
	comics := make([]*model.Comic, 0)
	cy := colly.NewCollector()
	cy.SetRequestTimeout(10 * time.Minute)
	cy.OnHTML(".cy_list_mh", func(e *colly.HTMLElement) {
		e.ForEach("ul", func(i int, item *colly.HTMLElement) {
			name := item.ChildText(".title")
			cover := item.ChildAttr(".pic > img", "src")
			detailUrl := fmt.Sprintf("%v", item.ChildAttr(".pic", "href"))
			comics = append(comics, &model.Comic{
				Name:     name,
				Cover:    cover,
				Origin:   a.Origin,
				Target:   fmt.Sprintf("%v%v", a.Origin, detailUrl),
				State:    0,
				OrderId:  a.OrderId,
				Language: a.Language,
			})
		})
	})
	err := cy.Visit(target)
	return comics, err
}

// Details 漫画详情 完毕
func (a *dongManLa) Details(comic *model.Comic) error {
	cy := colly.NewCollector()
	cy.SetRequestTimeout(10 * time.Minute)
	extensions.RandomUserAgent(cy)
	extensions.Referer(cy)
	cy.OnHTML("body", func(e *colly.HTMLElement) {
		e.DOM.Find(".cy_xinxi").Children().Each(func(i int, selection *goquery.Selection) {
			txt := selection.Text()
			if gstr.ContainsI(txt, "状态：") {
				stateStr := gstr.TrimLeft(txt, "状态：")
				if gstr.ContainsI(stateStr, "完结") {
					comic.State = 2
				}
				if gstr.ContainsI(stateStr, "连载") {
					comic.State = 1
				}
			}
			if gstr.ContainsI(txt, "类别：") {
				catalogue := gstr.TrimLeft(txt, "类别：")
				comic.Catalogue = gstr.ReplaceByMap(catalogue, map[string]string{
					",":    ",",
					" ":    "",
					"腐女漫画": "",
					"男男漫画": "",
				})
			}

			if gstr.ContainsI(txt, "作者：") {
				author := gstr.TrimLeft(txt, "作者：")
				comic.Author = author
			}

		})

		content := e.ChildText("#comic-description")
		comic.Content = gstr.Trim(gstr.ReplaceByMap(content, map[string]string{
			"\n": " ",
		}))

	})
	err := cy.Visit(comic.Target)
	return err
}

// ChapterCount 获取章节总数 完成
func (a *dongManLa) ChapterCount(comic *model.Comic) (int, error) {
	var err error
	var count int
	//这里要页面节点+请求数据整合
	cy := colly.NewCollector()
	cy.SetRequestTimeout(10 * time.Minute)
	extensions.RandomUserAgent(cy)
	extensions.Referer(cy)
	cy.OnHTML("#play_0 > #mh-chapter-list-ol-0", func(e *colly.HTMLElement) {
		count = e.DOM.Children().Size()
	})
	err = cy.Visit(comic.Target)
	return count, err
}

// Chapters 获取章节数列表 完成
func (a *dongManLa) Chapters(comic *model.Comic) ([]*model.Chapter, error) {
	var err error
	chapters := make([]*model.Chapter, 0)
	//这里要页面节点+请求数据整合
	cy := colly.NewCollector()
	cy.SetRequestTimeout(10 * time.Minute)
	extensions.RandomUserAgent(cy)
	extensions.Referer(cy)
	cy.OnHTML("#play_0 > #mh-chapter-list-ol-0", func(e *colly.HTMLElement) {
		e.ForEach("li", func(i int, element *colly.HTMLElement) {
			targer := element.ChildAttr("a", "href")
			//正则匹配
			regStr := `[0-9]+/1.html`
			reg := regexp.MustCompile(regStr)
			result := reg.FindString(targer)
			sort := gstr.ReplaceByMap(result, map[string]string{
				"/1.html": "",
			})
			chapters = append(chapters, &model.Chapter{
				Name:  element.DOM.Children().Find("p").Text(),
				Pid:   comic.UUID,
				State: 0,
				Sort:  gconv.Int(sort),
				Target: fmt.Sprintf("%v%v", a.Origin, gstr.ReplaceByMap(targer, map[string]string{
					a.Origin: "",
					"1.html": "all.html",
				})),
				Origin:  a.Origin,
				OrderId: a.OrderId,
			})
		})
	})
	err = cy.Visit(comic.Target)
	return chapters, err
}

// GetResource 漫画资源 完毕
func (a *dongManLa) GetResource(targetUrl string) ([]string, error) {
	chanTask := a.watchProgress(targetUrl)
	resource := make([]string, 0)
	cy := colly.NewCollector()
	cy.SetRequestTimeout(10 * time.Minute)
	cy.OnHTML(".imgListBox", func(e *colly.HTMLElement) {
		e.ForEach(".lazyBox", func(i int, item *colly.HTMLElement) {
			cover := item.ChildAttr("img", "data-src")
			resource = append(resource, cover)
		})

	})
	err := cy.Visit(targetUrl)
	chanTask <- true
	return resource, err
}
