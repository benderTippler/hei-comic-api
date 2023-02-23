package bank

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
	"github.com/gocolly/colly"
	"github.com/gocolly/colly/extensions"
	"github.com/gogf/gf/v2/container/gset"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/labstack/gommon/log"
	"hei-comic-api/app/collector/adapter"
	"hei-comic-api/app/model"
	"regexp"
	"strings"
	"time"
)

var KmWuAdapter = new(kmWu)

type kmWu struct {
	adapter.BaseAdapter
}

type chapterJson struct {
	Code    string `json:"vipCode"`
	Message string `json:"message"`
	Data    struct {
		List []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"list"`
	} `json:"data"`
}

// 初始化实例对象
func init() {
	err := KmWuAdapter.InitCfg(4) //初始化配置
	if err != nil {
		panic("KmWuAdapter InitCfg Fail")
	}
}

// StartUp 采集脚本入口
func (a *kmWu) StartUp() error {
	return nil
}

func (a *kmWu) GetUpdateComics() ([]*model.Comic, error) {
	comicSet := gset.New(true)
	comics := make([]*model.Comic, 0)
	page := 1
	for {
		target6Url := fmt.Sprintf("%v/rank/6-%v.html", a.Origin, page)
		tmp6Comics, err6 := a.List(target6Url)

		if err6 != nil {
			continue
		}
		if len(tmp6Comics) == 0 {
			break
		}
		for _, comic := range tmp6Comics {
			if !comicSet.Contains(comic.Target) {
				comics = append(comics, comic)
				comicSet.Add(comic.Target)
			}
		}
		if page >= 5 {
			break
		}
		page++
	}
	page = 1
	for {
		target5Url := fmt.Sprintf("%v/rank/5-%v.html", a.Origin, page)
		tmp5Comics, err5 := a.List(target5Url)
		if err5 != nil {
			continue
		}
		if len(tmp5Comics) == 0 {
			break
		}
		for _, comic := range tmp5Comics {
			if !comicSet.Contains(comic.Target) {
				comics = append(comics, comic)
				comicSet.Add(comic.Target)
			}
		}
		if page >= 5 {
			break
		}
		page++
	}
	comicSet.Clear()
	return comics, nil
}

// List 漫画列表页面 完结
func (a *kmWu) List(target string) ([]*model.Comic, error) {
	comics := make([]*model.Comic, 0)
	cy := colly.NewCollector()
	cy.SetRequestTimeout(10 * time.Minute)
	cy.OnHTML(".box > div > ul", func(e *colly.HTMLElement) {
		e.ForEach("li", func(i int, item *colly.HTMLElement) {
			name := item.ChildText("a:nth-child(1) > p:nth-child(2)")
			cover := item.ChildAttr(".card-graph > img", "data-src")
			detailUrl := fmt.Sprintf("%v%v", a.Origin, item.ChildAttr("a", "href"))
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

// Details 漫画详情 完毕
func (a *kmWu) Details(comic *model.Comic) error {
	cy := colly.NewCollector()
	cy.SetRequestTimeout(10 * time.Minute)
	extensions.RandomUserAgent(cy)
	extensions.Referer(cy)
	cy.OnHTML("body", func(e *colly.HTMLElement) {
		e.DOM.Find(".info > .tip").Children().Each(func(i int, selection *goquery.Selection) {
			if gstr.ContainsI(selection.Text(), "状态：") {
				stateStr := gstr.TrimLeft(selection.Text(), "状态：")
				if gstr.ContainsI(stateStr, "完结") {
					comic.State = 2
				}
				if gstr.ContainsI(stateStr, "连载") {
					comic.State = 1
				}
			}
			if gstr.ContainsI(selection.Text(), "题材：") {
				catalogue := gstr.TrimLeft(selection.Text(), "题材：")
				comic.Catalogue = gstr.ReplaceByMap(catalogue, map[string]string{
					"冒险热血": "冒险,热血",
					"武侠格斗": "武侠,格斗",
					"科幻魔幻": "科幻,魔幻",
					"侦探推理": "侦探,推理",
					"耽美爱情": "耽美,爱情",
					"生活漫画": "生活",
					"玄幻科幻": "玄幻,科幻",
				})
			}
		})
		txt := e.ChildText(".subtitle")
		if gstr.ContainsI(txt, "作者：") {
			author := gstr.TrimLeft(txt, "作者：")
			comic.Author = author
		}
		content := e.ChildText(".content")
		comic.Content = gstr.Trim(gstr.ReplaceByMap(content, map[string]string{
			"\n": " ",
		}))

	})
	err := cy.Visit(comic.Target)
	return err
}

// ChapterCount 获取章节总数 完毕
func (a *kmWu) ChapterCount(comic *model.Comic) (int, error) {
	var (
		count int
		err   error
	)
	//这里要页面节点+请求数据整合
	cy := colly.NewCollector()
	cy.SetRequestTimeout(10 * time.Minute)
	extensions.RandomUserAgent(cy)
	extensions.Referer(cy)
	cy.OnHTML("#chapterlistload > .view-win-list", func(e *colly.HTMLElement) {
		if e.DOM.Text() == "暂无章节" {
			count = 0
		} else {
			count = e.DOM.Children().Size()
		}
	})
	err = cy.Visit(comic.Target)

	//请求接口，显示剩余章节
	reg := regexp.MustCompile("\\/([1-9]\\d*)\\/$")
	result := reg.FindString(comic.Target)
	ajaxUrl := fmt.Sprintf("%v/chapterlist%v", a.Origin, result)
	client := g.Client()
	rep, err := client.Get(context.TODO(), ajaxUrl)
	defer rep.Close()
	body := rep.ReadAll()
	if len(body) == 0 {
		return count, err
	}
	if gstr.ContainsI(string(body), "Fatal error") {
		return count, nil
	}
	resultJson := new(chapterJson)
	if err = json.Unmarshal(body, resultJson); err != nil {
		return count, err
	}
	if resultJson.Code == "200" {
		count += len(resultJson.Data.List)
	}
	return count, err
}

// Chapters 获取章节数列表 完毕
func (a *kmWu) Chapters(comic *model.Comic) ([]*model.Chapter, error) {
	chapters := make([]*model.Chapter, 0)
	var err error
	//这里要页面节点+请求数据整合
	cy := colly.NewCollector()
	cy.SetRequestTimeout(10 * time.Minute)
	extensions.RandomUserAgent(cy)
	extensions.Referer(cy)
	cy.OnHTML("#chapterlistload > .view-win-list", func(e *colly.HTMLElement) {
		e.ForEach("li", func(i int, element *colly.HTMLElement) {
			targer := element.ChildAttr("a", "href")
			//正则匹配
			regestr := `(\d+)\.html`
			reg := regexp.MustCompile(regestr)
			result := reg.FindString(targer)
			sort := gstr.TrimRight(result, ".html")
			if element.Text != "暂无章节" {
				chapters = append(chapters, &model.Chapter{
					Name:    element.Text,
					Pid:     comic.UUID,
					State:   0,
					Sort:    gconv.Int(sort),
					Target:  fmt.Sprintf("%v%v", a.Origin, targer),
					Origin:  a.Origin,
					OrderId: a.OrderId,
				})
			}
		})
	})
	err = cy.Visit(comic.Target)
	if len(chapters) == 0 {
		return chapters, nil
	}
	//请求接口，显示剩余章节
	reg := regexp.MustCompile("\\/([1-9]\\d*)\\/$")
	result := reg.FindString(comic.Target)
	ajaxUrl := fmt.Sprintf("%v/chapterlist%v", a.Origin, result)
	client := g.Client()
	ctx := context.TODO()
	rep, err := client.Get(ctx, ajaxUrl)
	defer rep.Close()
	body := rep.ReadAll()
	if len(body) == 0 {
		return chapters, err
	}
	if gstr.ContainsI(string(body), "Fatal error") {
		return chapters, nil
	}
	resultJson := new(chapterJson)
	if err = json.Unmarshal(body, resultJson); err != nil {
		return chapters, err
	}
	if resultJson.Code == "200" {
		for _, v := range resultJson.Data.List {
			chapters = append(chapters, &model.Chapter{
				Name:    v.Name,
				Pid:     comic.UUID,
				State:   0,
				Sort:    gconv.Int(v.ID),
				Target:  fmt.Sprintf("%v%v%v.html", a.Origin, result, v.ID),
				Origin:  a.Origin,
				OrderId: a.OrderId,
			})
		}
	}
	return chapters, err
}

// GetResource 漫画资源 完毕
func (a *kmWu) GetResource(targetUrl string) ([]string, error) {

	var err error
	resource := make([]string, 0)
	// 禁用chrome headless
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("blink-settings", "imagesEnabled=false"),
		chromedp.Flag("disk-cache-dir", a.CachePath),
	)
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()
	// create chrome instance
	ctx, cancel := chromedp.NewContext(
		allocCtx,
		chromedp.WithLogf(log.Printf),
	)
	defer cancel()

	// create a timeout
	ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var nodes string
	err = chromedp.Run(ctx,
		chromedp.Navigate(targetUrl),
		chromedp.WaitVisible(`.chapter-img-box`),
		chromedp.OuterHTML(`.main_img`, &nodes, chromedp.ByQuery),
	)
	dom, err := goquery.NewDocumentFromReader(strings.NewReader(nodes))
	if err != nil {
		return resource, err
	}
	dom.Find(".main_img").Children().Each(func(i int, selection *goquery.Selection) {
		var imgStr string
		img, _ := selection.Find("img").Attr("src")
		if img == "/static/images/load.gif" {
			imgStr, _ = selection.Find("img").Attr("data-src")
		} else {
			imgStr = img
		}
		resource = append(resource, imgStr)
	})
	return resource, err
}
