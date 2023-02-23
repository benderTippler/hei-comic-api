package adapter

import (
	"context"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/dop251/goja"
	"github.com/gocolly/colly"
	"github.com/gocolly/colly/extensions"
	"github.com/gogf/gf/v2/container/gvar"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/gclient"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
	"hei-comic-api/app/model"
	"regexp"
	"strings"
	"time"
)

var ManHuaAdapter = new(manHua)

type manHua struct {
	BaseAdapter
}

// 初始化 mangabz 实例对象
func init() {
	err := ManHuaAdapter.InitCfg(2) //初始化配置
	if err != nil {
		panic("ManHuaAdapter InitCfg Fail")
	}
}

// StartUp 采集脚本入口
func (a *manHua) StartUp() error {
	return nil
}

func (a *manHua) GetUpdateComics() ([]*model.Comic, error) {
	//comicSet := gset.New(true)
	comics := make([]*model.Comic, 0)
	page := 1
	for {
		targetUrl := fmt.Sprintf("%v/update_%v.shtml", a.Origin, page)
		client := g.Client()
		rsp, err := client.Get(context.TODO(), targetUrl)
		defer rsp.Close()
		if err != nil {
			return comics, nil
		}

		dom, err := goquery.NewDocumentFromReader(strings.NewReader(rsp.ReadAllString()))
		if err != nil {
			break
		}
		count := dom.Find(".newpic_bg > .boxdiv1").Size()
		if count == 0 {
			fmt.Println("终止for循环")
			break
		}
		dom.Find(".boxdiv1").Each(func(i int, selection *goquery.Selection) {
			parent := selection.Children()
			var target string
			href, _ := parent.Find(".picborder > a").Attr("href")
			cover, _ := parent.Find(".picborder > a >img").Attr("src")
			if gstr.ContainsI(href, "http") {
				target = href
			} else {
				target = fmt.Sprintf("%v/%v", a.Origin, href)
			}

			name := parent.Find(".pictextst").Text()
			comics = append(comics, &model.Comic{
				Name:     name,
				Cover:    cover,
				Origin:   a.Origin,
				Target:   target,
				Language: a.Language,
				OrderId:  a.OrderId,
			})
		})
		page++
	}
	return comics, nil
}

// List 漫画列表页面 完毕
func (a *manHua) List(target string) ([]*model.Comic, error) {
	comicList := make([]*model.Comic, 0)
	var rep *gclient.Response
	var err error
	client := g.Client()
	rep, err = client.Get(context.TODO(), target)
	defer rep.Close()
	if err != nil {
		return comicList, err
	}
	callback := rep.Request.URL.Query().Get("callback")
	bodyStr := rep.ReadAllString()
	bodyStr = gstr.TrimLeft(gstr.TrimLeft(gstr.TrimRight(bodyStr, ");"), callback), "(")
	jsonResult := gjson.New(bodyStr)

	var list *gvar.Var
	list = jsonResult.Get("result")
	for _, v := range list.Maps() {
		var targetUrl string = v["comic_url"].(string)
		if !gstr.ContainsI(targetUrl, "http") {
			targetUrl = fmt.Sprintf("%v%v", a.Origin, v["comic_url"].(string))
		}
		comic := &model.Comic{
			Name:     v["name"].(string),
			State:    0,
			Origin:   a.Origin,
			Target:   targetUrl, // 包括域名
			OrderId:  a.OrderId,
			Language: a.Language,
		}
		comicList = append(comicList, comic)
	}
	return comicList, nil
}

// Details 漫画详情 完毕
func (a *manHua) Details(comic *model.Comic) error {
	var err error
	isOld := gstr.ContainsI(comic.Target, "manhua.dmzj.com")
	if isOld {
		url := comic.Target
		cy := colly.NewCollector()
		cy.SetRequestTimeout(10 * time.Minute)
		extensions.RandomUserAgent(cy)
		extensions.Referer(cy)
		cy.OnHTML("body", func(e *colly.HTMLElement) {
			author := e.ChildText(".anim-main_list > table:nth-child(1) > tbody:nth-child(1) > tr:nth-child(3) > td:nth-child(2)")
			stateStr := e.ChildText(".anim-main_list > table:nth-child(1) > tbody:nth-child(1) > tr:nth-child(5) > td:nth-child(2)")
			catalogue := make([]string, 0)
			e.ForEach(".anim-main_list > table:nth-child(1) > tbody:nth-child(1) > tr:nth-child(7) > td:nth-child(2) > a", func(i int, element *colly.HTMLElement) {
				catalogue = append(catalogue, gstr.Trim(element.Text))
			})
			if gstr.ContainsI(stateStr, "完结") {
				comic.State = 2
			}
			if gstr.ContainsI(stateStr, "连载") {
				comic.State = 1
			}
			comic.Author = author
			comic.Catalogue = gstr.Join(catalogue, ",")
			content := e.ChildText(".line_height_content")
			comic.Content = gstr.Trim(gstr.ReplaceByMap(content, map[string]string{
				"\n": " ",
			}))
			comic.Cover = e.ChildAttr("#cover_pic", "src")
		})
		err = cy.Visit(url)
	} else { //新版数据
		url := comic.Target
		cy := colly.NewCollector()
		cy.SetRequestTimeout(10 * time.Minute)
		extensions.RandomUserAgent(cy)
		extensions.Referer(cy)
		cy.OnHTML("body", func(e *colly.HTMLElement) {
			e.DOM.Find(".comic_deCon_liO").Children().Each(func(i int, selection *goquery.Selection) {
				txt := selection.Text()
				if gstr.ContainsI(txt, "作者：") {
					author := gstr.TrimLeft(txt, "作者：")
					comic.Author = author
				}
				if gstr.ContainsI(txt, "状态：") {
					stateStr := gstr.TrimLeft(txt, "状态：")
					if gstr.ContainsI(stateStr, "完结") {
						comic.State = 2
					}
					if gstr.ContainsI(stateStr, "连载") {
						comic.State = 1
					}
				}
				if gstr.ContainsI(txt, "类型：") {
					catalogue := gstr.TrimLeft(txt, "类型：")
					comic.Catalogue = gstr.ReplaceByMap(catalogue, map[string]string{
						"|": ",",
					})
				}

			})
			content := e.ChildText(".comic_deCon_d")
			comic.Content = gstr.Trim(gstr.ReplaceByMap(content, map[string]string{
				"\n": " ",
			}))
			comic.Cover = e.ChildAttr(".comic_i_img > a >img", "src")
		})
		err = cy.Visit(url)
	}
	return err
}

// ChapterCount 获取章节总数 完毕
func (a *manHua) ChapterCount(comic *model.Comic) (int, error) {
	var err error
	var count int
	isOld := gstr.ContainsI(comic.Target, "manhua.dmzj.com")
	if isOld {
		cy := colly.NewCollector()
		cy.SetRequestTimeout(10 * time.Minute)
		extensions.RandomUserAgent(cy)
		extensions.Referer(cy)
		cy.OnHTML("body", func(e *colly.HTMLElement) {
			e.DOM.Children().Find(".cartoon_online_border > ul").Each(func(i int, selection *goquery.Selection) {
				count += selection.Children().Size()
			})
		})
		err = cy.Visit(comic.Target)
	} else {
		cy := colly.NewCollector()
		cy.SetRequestTimeout(10 * time.Minute)
		extensions.RandomUserAgent(cy)
		extensions.Referer(cy)
		cy.OnHTML("body", func(e *colly.HTMLElement) {
			e.DOM.Children().Find(".tab-content-selected > .list_con_li").Each(func(i int, selection *goquery.Selection) {
				count += selection.Children().Size()
			})
		})
		err = cy.Visit(comic.Target)
	}
	return count, err
}

// Chapters 获取章节数列表 完毕
func (a *manHua) Chapters(comic *model.Comic) ([]*model.Chapter, error) {
	chapters := make([]*model.Chapter, 0)
	var err error
	isOld := gstr.ContainsI(comic.Target, "manhua.dmzj.com")
	if isOld {
		cy := colly.NewCollector()
		cy.SetRequestTimeout(10 * time.Minute)
		extensions.RandomUserAgent(cy)
		extensions.Referer(cy)
		cy.OnHTML("body", func(e *colly.HTMLElement) {
			e.DOM.Children().Find(".cartoon_online_border > ul > li").Each(func(i int, selection *goquery.Selection) {
				targer, _ := selection.Children().Attr("href")
				//正则匹配
				regestr := `(\d+)\.shtml`
				reg := regexp.MustCompile(regestr)
				result := reg.FindString(targer)
				sort := gstr.TrimRight(result, ".shtml")
				chapters = append(chapters, &model.Chapter{
					Name:    selection.Children().Text(),
					Pid:     comic.UUID,
					State:   0,
					Sort:    gconv.Int(sort),
					Target:  fmt.Sprintf("%v%v", a.Origin, targer),
					Origin:  a.Origin,
					OrderId: a.OrderId,
				})
			})
		})
		err = cy.Visit(comic.Target)
	} else {
		cy := colly.NewCollector()
		cy.SetRequestTimeout(10 * time.Minute)
		extensions.RandomUserAgent(cy)
		extensions.Referer(cy)
		cy.OnHTML(".tab-content-selected > .list_con_li", func(e *colly.HTMLElement) {
			e.ForEach("li", func(i int, element *colly.HTMLElement) {
				targer := element.ChildAttr("a", "href")
				//正则匹配
				regestr := `(\d+)\.html`
				reg := regexp.MustCompile(regestr)
				result := reg.FindString(targer)
				sort := gstr.TrimRight(result, ".html")
				chapters = append(chapters, &model.Chapter{
					Name:    element.ChildText("a"),
					Pid:     comic.UUID,
					State:   0,
					Sort:    gconv.Int(sort),
					Target:  targer,
					Origin:  a.Origin,
					OrderId: a.OrderId,
				})
			})
		})
		err = cy.Visit(comic.Target)
	}
	return chapters, err
}

// GetResource 漫画资源 完毕
func (a *manHua) GetResource(targetUrl string) ([]string, error) {
	resource := make([]string, 0)
	var ImageList = make([]interface{}, 0)
	var err error
	isOld := gstr.ContainsI(targetUrl, "manhua.dmzj.com")
	if isOld {
		cy := colly.NewCollector()
		cy.OnHTML("head > script:nth-child(5)", func(e *colly.HTMLElement) {
			all := e.DOM.Text()
			all += `
	let obj = eval(pages)
    let List = new Array()
    obj.forEach(function (value, index) {
        List.push("https://images.dmzj.com/"+value)
    })
`
			vm := goja.New()
			_, err = vm.RunString(all)
			if err != nil {
				return
			}
			ImageList = vm.Get("List").Export().([]interface{})
			for _, value := range ImageList {
				resource = append(resource, value.(string))
			}
		})
		err = cy.Visit(targetUrl)
	} else {
		cy := colly.NewCollector()
		cy.OnHTML("head > script:nth-child(11)", func(e *colly.HTMLElement) {
			all := e.DOM.Text()
			all += `
    var img_prefix = 'https://images.dmzj.com/';
    pages = pages.replace(/\n/g,"");
    pages = pages.replace(/\r/g,"|");
    var info = eval("(" + pages + ")");
    var imageS = (info["page_url"].split('|'))
    var List = new Array()
    imageS.forEach(function(element){
        List.push(img_prefix+element)
    })
`
			vm := goja.New()
			vm.RunString(all)
			ImageList = vm.Get("List").Export().([]interface{})
			for _, value := range ImageList {
				resource = append(resource, value.(string))
			}

		})
		err = cy.Visit(targetUrl)
	}
	return resource, err
}
