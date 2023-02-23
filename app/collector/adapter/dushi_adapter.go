package adapter

import (
	"context"
	"fmt"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/dop251/goja"
	"github.com/gocolly/colly"
	"github.com/gocolly/colly/extensions"
	"github.com/gocolly/colly/proxy"
	"github.com/gogf/gf/v2/container/gset"
	"github.com/gogf/gf/v2/text/gstr"
	"hei-comic-api/app/model"
	"io/ioutil"
	"time"
)

var DuShiAdapter = new(duShi)

type duShi struct {
	BaseAdapter
}

// 初始化实例对象
func init() {
	err := DuShiAdapter.InitCfg(9) //初始化配置
	if err != nil {
		panic("DuShiAdapter InitCfg Fail")
	}

}

// StartUp 采集脚本入口
func (a *duShi) StartUp() error {
	return nil
}

func (a *duShi) GetUpdateComics() ([]*model.Comic, error) {
	comicSet := gset.New(true)
	comics := make([]*model.Comic, 0)
	var page int = 1
	for {
		url := fmt.Sprintf("%v/update/%v/", a.Origin, page)
		list, _ := a.List(url)
		if page > 10 || len(list) == 0 {
			break
		}
		for _, v := range list {
			if !comicSet.Contains(v.Target) {
				comics = append(comics, v)
				comicSet.Add(v.Target)
			}
		}
		page++
	}
	comicSet.Clear()
	return comics, nil
}

// List 漫画列表页面 完成
func (a *duShi) List(target string) ([]*model.Comic, error) {
	var err error
	comics := make([]*model.Comic, 0)
	cy := colly.NewCollector()
	cy.SetRequestTimeout(10 * time.Minute)
	extensions.RandomUserAgent(cy)
	extensions.Referer(cy)
	cy.OnRequest(func(request *colly.Request) {
		request.Headers.Set("Host", a.Origin)
	})
	cy.OnHTML("#contList", func(e *colly.HTMLElement) {
		e.ForEach("li", func(i int, item *colly.HTMLElement) {
			name := item.ChildText(".ell")
			cover := item.ChildAttr(".cover > img", "src")
			detailUrl := item.ChildAttr(".cover", "href")
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
	err = cy.Visit(target)
	return comics, err
}

// Details 漫画详情 完成, 因为列表数据已经补充完整
func (a *duShi) Details(comic *model.Comic) error {
	cy := colly.NewCollector()
	cy.SetRequestTimeout(10 * time.Minute)
	extensions.RandomUserAgent(cy)
	extensions.Referer(cy)
	//cy.OnRequest(func(request *colly.Request) {
	//	request.Headers.Set("Host", a.Origin)
	//})
	cy.OnHTML(".comic-view", func(e *colly.HTMLElement) {
		stateStr := e.ChildText(".detail-list > li:nth-child(1) > span:nth-child(1) > a:nth-child(2)")
		if gstr.ContainsI(stateStr, "已完结") {
			comic.State = 2
		}
		if gstr.ContainsI(stateStr, "连载中") {
			comic.State = 1
		}
		//漫画类型
		catalogue := e.ChildText(".breadcrumb-bar > ol:nth-child(1) > li:nth-child(5)")
		comic.Catalogue = gstr.ReplaceByMap(catalogue, map[string]string{
			",": ",",
			"/": "",
		})
		//作者
		comic.Author = e.ChildText(".detail-list > li:nth-child(2) > span:nth-child(2) > a:nth-child(2)")
		content := e.ChildText("#intro-all")
		comic.Content = gstr.Trim(gstr.ReplaceByMap(content, map[string]string{
			"\n":    "",
			"漫画简介：": "",
		}))
	})
	err := cy.Visit(comic.Target)
	return err
}

// ChapterCount 获取章节总数 完成
func (a *duShi) ChapterCount(comic *model.Comic) (int, error) {
	var (
		count int
		err   error
	)
	//这里要页面节点+请求数据整合
	cy := colly.NewCollector()
	cy.SetRequestTimeout(10 * time.Minute)
	extensions.RandomUserAgent(cy)
	extensions.Referer(cy)
	cy.OnHTML("#chapter-list-1", func(e *colly.HTMLElement) {
		count = e.DOM.Children().Size()
	})
	err = cy.Visit(comic.Target)
	return count, err
}

// Chapters 获取章节数列表
func (a *duShi) Chapters(comic *model.Comic) ([]*model.Chapter, error) {
	var err error
	chapters := make([]*model.Chapter, 0)
	//这里要页面节点+请求数据整合
	cy := colly.NewCollector()
	cy.SetRequestTimeout(10 * time.Minute)
	extensions.RandomUserAgent(cy)
	extensions.Referer(cy)
	cy.OnHTML("#chapter-list-1", func(e *colly.HTMLElement) {
		e.ForEach("li", func(i int, element *colly.HTMLElement) {
			targer := element.ChildAttr("a", "href")
			name := element.ChildText("a")
			//正则匹配
			chapters = append(chapters, &model.Chapter{
				Name:    name,
				Pid:     comic.UUID,
				State:   0,
				Sort:    i + 1,
				Target:  fmt.Sprintf("%v%v", a.Origin, targer),
				Origin:  a.Origin,
				OrderId: a.OrderId,
			})
		})
	})
	err = cy.Visit(comic.Target)
	return chapters, err
}

// GetResource 漫画资源 完成
func (a *duShi) GetResource(targetUrl string) ([]string, error) {
	var err error
	resource := make([]string, 0)
	cy := colly.NewCollector()
	cy.SetRequestTimeout(10 * time.Minute)
	extensions.RandomUserAgent(cy)
	extensions.Referer(cy)
	//配置两个代理
	if a.MaxTry < 5 && false {
		//ips := utills.GetProxy()
		rp, err := proxy.RoundRobinProxySwitcher("http://47.101.44.122:80")
		if err != nil {
			return resource, err
		}
		cy.SetProxyFunc(rp)
	}
	cy.OnRequest(func(request *colly.Request) {
		request.Headers.Set("Host", a.Origin)
	})

	cy.OnHTML("body > script:nth-child(1)", func(e *colly.HTMLElement) {
		jsTxt := e.Text
		vm2 := goja.New()
		_, err = vm2.RunString(jsTxt)
		if err != nil {
			fmt.Println("m2.RunString", err)
			return
		}
		if jsTxt == "" {
			fmt.Println("js")
			return
		}

		if !gstr.ContainsI(jsTxt, "chapterImages") {
			return
		}
		list := vm2.Get("chapterImages").Export()
		for _, v := range list.([]interface{}) {
			resource = append(resource, v.(string))
		}
	})
	err = cy.Visit(targetUrl)
	return resource, err
}

// chromedp 无头浏览器封装
func (a *duShi) chromedp(target string) (string, error) {
	// 禁用chrome headless
	context.TODO()
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
		chromedp.Flag("blink-settings", "imagesEnabled=false"),
		chromedp.UserAgent(`Mozilla/5.0 (Windows NT 6.3; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/73.0.3683.103 Safari/537.36`),
	)
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()
	// create chrome instance
	ctx, cancel := chromedp.NewContext(
		allocCtx,
	)
	defer cancel()

	// create a timeout
	ctx, cancel = context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	var nodes string
	err := chromedp.Run(ctx, chromeTask(target))
	return nodes, err
}

// 采集任务
func chromeTask(target string) chromedp.Tasks {
	return chromedp.Tasks{
		// 1、单开一个页面，保存cookie数据
		chromedp.Navigate(target),
		// 2、获取cookie
		saveCookies(),
	}
}

// 保存Cookies
func saveCookies() chromedp.ActionFunc {
	return func(ctx context.Context) (err error) {
		// 等待二维码登陆
		chromedp.WaitVisible(`#images`)
		// cookies的获取对应是在devTools的network面板中
		// 1. 获取cookies
		cookies, err := network.GetAllCookies().Do(ctx)
		if err != nil {
			return
		}

		// 2. 序列化
		cookiesData, err := network.GetAllCookiesReturns{Cookies: cookies}.MarshalJSON()
		if err != nil {
			return
		}

		// 3. 存储到临时文件
		if err = ioutil.WriteFile("cookies.tmp", cookiesData, 0755); err != nil {
			return
		}
		return
	}
}
