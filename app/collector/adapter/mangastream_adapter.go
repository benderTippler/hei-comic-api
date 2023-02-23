package adapter

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
	"hei-comic-api/app/model"
	"regexp"
	"time"
)

var MangasTreamAdapter = new(mangasTream)

type mangasTream struct {
	BaseAdapter
}

// 初始化包子实例对象
func init() {
	err := MangasTreamAdapter.InitCfg(7) //初始化配置
	if err != nil {
		panic("BaoZiAdapter InitCfg Fail")
	}
}

// StartUp 采集器启动
func (a *mangasTream) StartUp() error {
	return nil
}

func (a *mangasTream) GetUpdateComics() ([]*model.Comic, error) {
	comics := make([]*model.Comic, 0)
	page := 1
	for {
		targetUrl := fmt.Sprintf("%v/latest-manga?page=%v", a.Origin, page)
		tmpComic, err := a.updateList(targetUrl)
		if err != nil {
			continue
		}
		if len(tmpComic) == 0 || page >= 4 {
			break
		}
		comics = append(comics, tmpComic...)
		page++
	}
	return comics, nil
}

func (a *mangasTream) updateList(target string) ([]*model.Comic, error) {
	var err error
	comics := make([]*model.Comic, 0)
	cy := colly.NewCollector()
	cy.SetRequestTimeout(10 * time.Minute)
	cy.OnHTML(".cate-manga", func(e *colly.HTMLElement) {
		e.ForEach(".col-md-6 > .media", func(i int, item *colly.HTMLElement) {
			name := item.ChildText(".media-body > a")
			cover := item.ChildAttr(".media-object", "src")
			detailUrl := item.ChildAttr(".media-body > a", "href")
			covers := gstr.Split(cover, "https")
			if len(covers) == 2 {
				cover = "https" + covers[1]
			}

			comics = append(comics, &model.Comic{
				Name:     name,
				Cover:    cover,
				Origin:   a.Origin,
				OrderId:  a.OrderId,
				Language: a.Language,
				Target:   detailUrl,
				State:    0,
			})
		})
	})
	err = cy.Visit(target)
	return comics, err
}

// List 漫画列表页面 完成
func (a *mangasTream) List(target string) ([]*model.Comic, error) {
	var err error
	comics := make([]*model.Comic, 0)
	cy := colly.NewCollector()
	cy.SetRequestTimeout(10 * time.Minute)
	cy.OnHTML(".col-md-8 > div:nth-child(2)", func(e *colly.HTMLElement) {
		e.ForEach(".col-md-6.col-sm-6", func(i int, item *colly.HTMLElement) {
			name := item.ChildText(".manga-newest")
			cover := item.ChildAttr(".media-object", "src")
			detailUrl := item.ChildAttr(".media-body > a", "href")
			covers := gstr.Split(cover, "https")
			if len(covers) == 2 {
				cover = "https" + covers[1]
			}
			comics = append(comics, &model.Comic{
				Name:     name,
				Cover:    cover,
				Origin:   a.Origin,
				OrderId:  a.OrderId,
				Language: a.Language,
				Target:   detailUrl,
				State:    0,
			})
		})
	})
	err = cy.Visit(target)
	return comics, err
}

// Details 漫画详情 完成
func (a *mangasTream) Details(comic *model.Comic) error {
	var err error
	cy := colly.NewCollector()
	cy.SetRequestTimeout(10 * time.Minute)
	cy.OnHTML(".bodycontainer", func(e *colly.HTMLElement) {
		description := e.DOM.Children().Find(".description-update")
		html, err := description.Html()
		if err != nil {
			return
		}
		labelSlice := gstr.Split(html, "<br/>")
		for _, label := range labelSlice {
			label = trimHtml(label)
			if gstr.ContainsI(label, "Author(s):") { //作者信息
				author := gstr.Trim(label, "Author(s):")
				comic.Author = author
			}
			if gstr.ContainsI(label, "Genre:") { // 动漫类型
				catalogue := gstr.ReplaceByMap(gstr.Trim(label, "Genre:"), map[string]string{
					"，":  ",",
					"\n": "",
				})
				comic.Catalogue = gstr.ToLower(gstr.TrimRight(catalogue, ","))
			}
			if gstr.ContainsI(label, "Status:") { // 动漫更新状态
				status := gstr.ToLower(gstr.Trim(label, "Status:"))
				if status == "ongoing" {
					comic.State = 1
				} else if status == "completed" {
					comic.State = 2
				}
			}

		}

		// 获取内容
		content := e.DOM.Children().Find("#example2").Text()
		comic.Content = gstr.ReplaceByMap(content, map[string]string{
			"\n": "",
		})
	})
	err = cy.Visit(comic.Target)
	return err
}

// ChapterCount 获取章节总数 完成
func (a *mangasTream) ChapterCount(comic *model.Comic) (int, error) {
	count := 0
	var err error
	cy := colly.NewCollector()
	cy.SetRequestTimeout(10 * time.Minute)
	cy.OnHTML("#examples", func(e *colly.HTMLElement) {
		count = e.DOM.Children().Find(".chapter-list > ul > .row").Size()
	})
	err = cy.Visit(comic.Target)
	return count, err
}

// Chapters 获取章节数列表
func (a *mangasTream) Chapters(comic *model.Comic) ([]*model.Chapter, error) {
	var err error
	cy := colly.NewCollector()
	chapters := make([]*model.Chapter, 0)
	cy.OnHTML("#examples", func(e *colly.HTMLElement) {
		e.DOM.Children().Find(".chapter-list > ul > .row").Each(func(i int, item *goquery.Selection) {
			name, _ := item.Children().Find("a").Attr("title")
			target, _ := item.Children().Find("a").Attr("href")
			regestr := "[1-9][0-9]*([\\.][0-9]{1,2})?"
			reg := regexp.MustCompile(regestr)
			result := reg.FindString(name)
			sort := gstr.ReplaceByMap(result, map[string]string{})
			chapters = append(chapters, &model.Chapter{
				Name:    fmt.Sprintf("chapter %v", sort),
				Pid:     comic.UUID,
				State:   0,
				Sort:    gconv.Int(sort),
				Target:  target,
				Origin:  a.Origin,
				OrderId: a.OrderId,
			})
		})
	})
	err = cy.Visit(comic.Target)
	return chapters, err
}

// GetResource 漫画资源 完成
func (a *mangasTream) GetResource(targetUrl string) ([]string, error) {
	var err error
	resource := make([]string, 0)
	cy := colly.NewCollector()
	cy.SetRequestTimeout(30 * time.Second)
	cy.OnHTML("#arraydata", func(e *colly.HTMLElement) {
		resource = gstr.Explode(",", e.Text)
	})
	cy.Visit(targetUrl)
	return resource, err
}
