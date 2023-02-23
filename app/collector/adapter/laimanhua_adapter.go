package adapter

import (
	"fmt"
	"github.com/dop251/goja"
	"github.com/gocolly/colly"
	"github.com/gocolly/colly/extensions"
	"github.com/gogf/gf/v2/container/gset"
	"github.com/gogf/gf/v2/encoding/gbase64"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
	"hei-comic-api/app/model"
	"hei-comic-api/app/utills"
	"regexp"
	"sort"
	"sync"
	"time"
)

var LaiManHuaAdapter = new(laiManHua)

type laiManHua struct {
	BaseAdapter
}

// 初始化实例对象
func init() {
	err := LaiManHuaAdapter.InitCfg(8) //初始化配置
	if err != nil {
		panic("LaiManHuaAdapter InitCfg Fail")
	}

}

// StartUp 采集脚本入口
func (a *laiManHua) StartUp() error {
	return nil
}

func (a *laiManHua) GetUpdateComics() ([]*model.Comic, error) {
	comicSet := gset.New(true)
	comics := make([]*model.Comic, 0)
	wgL := sync.WaitGroup{}
	pageChan := make(chan bool, 2)
	for _, targetTpl := range a.ListTemplate {
		wgL.Add(1)
		pageChan <- true
		go func(targetTpl string) {
			defer wgL.Done()
			var page int = 1
			for {
				url := fmt.Sprintf(targetTpl, a.Origin, page)
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
			<-pageChan
		}(targetTpl)
	}
	wgL.Wait()
	comicSet.Clear()
	return comics, nil
}

// List 漫画列表页面 完成
func (a *laiManHua) List(target string) ([]*model.Comic, error) {
	var err error
	comics := make([]*model.Comic, 0)
	cy := colly.NewCollector()
	cy.SetRequestTimeout(10 * time.Minute)
	cy.OnHTML("#dmList > ul", func(e *colly.HTMLElement) {
		e.ForEach("li", func(i int, item *colly.HTMLElement) {
			name := utills.ConvertToString(item.ChildAttr("dl>dt>a", "title"), "gbk", "utf-8")
			cover := item.ChildAttr(".pic > img", "src")
			detailUrl := fmt.Sprintf("%v", item.ChildAttr(".pic", "href"))
			content := utills.ConvertToString(item.ChildText(".intro"), "gbk", "utf-8")

			var state int
			stateStr := utills.ConvertToString(item.ChildText("dl:nth-child(2) > dd:nth-child(2) > p:nth-child(2) > span:nth-child(2)"), "gbk", "utf-8")
			if gstr.ContainsI(stateStr, "完结") {
				state = 2
			}
			if gstr.ContainsI(stateStr, "连载") {
				state = 1
			}

			catalogueStr := utills.ConvertToString(item.ChildText("dl:nth-child(2) > dd:nth-child(2) > p:nth-child(3) > a:nth-child(2)"), "gbk", "utf-8")
			catalogue := gstr.ReplaceByMap(catalogueStr, map[string]string{
				"少年热血": "热血",
				"武侠格斗": "武侠,格斗",
				"科幻魔幻": "科幻,魔幻",
				"竞技体育": "竞技,体育",
				"爆笑喜剧": "爆笑,喜剧",
				"侦探推理": "侦探,推理",
				"恐怖灵异": "恐怖,灵异",
				"耽美人生": "耽美,生活",
				"恋爱生活": "恋爱,生活",
				"生活漫画": "生活",
				"故事漫画": "故事",
				"漫画":   "",
				"少女爱情": "恋爱",
				"其他漫画": "其他",
				"百合女性": "百合",
			})
			content = gstr.Trim(gstr.ReplaceByMap(content, map[string]string{
				"简　介：": " ",
			}))
			comics = append(comics, &model.Comic{
				Name:      name,
				Cover:     cover,
				Origin:    a.Origin,
				Target:    fmt.Sprintf("%v%v", a.Origin, detailUrl),
				State:     state,
				OrderId:   a.OrderId,
				Language:  a.Language,
				Content:   content,
				Catalogue: catalogue,
			})
		})
	})
	err = cy.Visit(target)
	return comics, err
}

// Details 漫画详情 完成, 因为列表数据已经补充完整
func (a *laiManHua) Details(comic *model.Comic) error {
	var err error
	cy := colly.NewCollector()
	cy.SetRequestTimeout(10 * time.Minute)
	cy.OnHTML("p.w260:nth-child(2)", func(e *colly.HTMLElement) {
		author := gstr.ReplaceByMap(utills.ConvertToString(e.DOM.Text(), "gbk", "utf-8"), map[string]string{
			"原著作者：": "",
		})
		comic.Author = author
	})
	err = cy.Visit(comic.Target)
	return err
}

// ChapterCount 获取章节总数 完成
func (a *laiManHua) ChapterCount(comic *model.Comic) (int, error) {
	var (
		count int
		err   error
	)
	//这里要页面节点+请求数据整合
	cy := colly.NewCollector()
	cy.SetRequestTimeout(10 * time.Minute)
	extensions.RandomUserAgent(cy)
	extensions.Referer(cy)
	cy.OnHTML("#play_0 > ul", func(e *colly.HTMLElement) {
		count = e.DOM.Children().Size()
	})
	err = cy.Visit(comic.Target)
	return count, err
}

// Chapters 获取章节数列表
func (a *laiManHua) Chapters(comic *model.Comic) ([]*model.Chapter, error) {
	var err error
	chapters := make([]*model.Chapter, 0)
	//这里要页面节点+请求数据整合
	cy := colly.NewCollector()
	cy.SetRequestTimeout(10 * time.Minute)
	extensions.RandomUserAgent(cy)
	extensions.Referer(cy)
	cy.OnHTML("#play_0 > ul", func(e *colly.HTMLElement) {
		e.ForEach("li", func(i int, element *colly.HTMLElement) {
			targer := element.ChildAttr("a", "href")
			name := utills.ConvertToString(element.ChildAttr("a", "title"), "gbk", "utf-8")
			//正则匹配
			chapters = append(chapters, &model.Chapter{
				Name:    name,
				Pid:     comic.UUID,
				State:   0,
				Sort:    i,
				Target:  fmt.Sprintf("%v%v", a.Origin, targer),
				Origin:  a.Origin,
				OrderId: a.OrderId,
			})
		})
	})
	err = cy.Visit(comic.Target)
	//排序处理
	sort.SliceStable(chapters, func(i, j int) bool {
		return chapters[i].Sort > chapters[j].Sort
	})
	for i, v := range chapters {
		v.Sort = i + 1
	}
	return chapters, err
}

// GetResource 漫画资源 完成
func (a *laiManHua) GetResource(targetUrl string) ([]string, error) {
	var err error
	resource := make([]string, 0)
	regStr := `[0-9]+.html$`
	reg := regexp.MustCompile(regStr)
	result := reg.FindString(targetUrl)
	currentChapterid := gconv.Int(gstr.ReplaceByMap(result, map[string]string{
		".html": "",
	}))
	fmt.Println(currentChapterid)
	//这里要页面节点+请求数据整合
	cy := colly.NewCollector()
	cy.SetRequestTimeout(10 * time.Minute)
	extensions.RandomUserAgent(cy)
	extensions.Referer(cy)
	cy.OnHTML("body > script:nth-child(21)", func(e *colly.HTMLElement) {
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

		if !gstr.ContainsI(jsTxt, "picTree") {
			return
		}

		picTree := vm2.Get("picTree").Export()
		var prefix string = "https://mhpic6.kingwar.cn"
		if currentChapterid > 542724 {
			prefix = "https://mhpic5.kingwar.cn"
		}
		if picTree.(string) == "" {
			return
		}
		listStr, err := gbase64.DecodeToString(picTree.(string))
		if err != nil {
			fmt.Println("gbase64.DecodeToString", err)
			return
		}
		list := gstr.Split(listStr, "$qingtiandy$")
		for _, v := range list {
			resource = append(resource, fmt.Sprintf("%v%v", prefix, v))
		}
	})
	err = cy.Visit(targetUrl)
	return resource, err
}
