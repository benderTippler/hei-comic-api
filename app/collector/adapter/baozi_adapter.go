package adapter

import (
	"context"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
	"github.com/gogf/gf/v2/container/gset"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/labstack/gommon/log"
	"hei-comic-api/app/model"
	chromedp2 "hei-comic-api/base/chromedp"
	"regexp"
	"strings"
	"time"
)

var BaoZiAdapter = new(baoZi)

type baoZi struct {
	BaseAdapter
}

// 初始化包子实例对象
func init() {
	err := BaoZiAdapter.InitCfg(1) //初始化配置
	if err != nil {
		panic("BaoZiAdapter InitCfg Fail")
	}
}

// StartUp 采集器启动
func (a *baoZi) StartUp() error {
	return nil
}

func (a *baoZi) GetUpdateComics() ([]*model.Comic, error) {
	comicSet := gset.New(true)
	comics := make([]*model.Comic, 0)
	targetUrl := fmt.Sprintf("%v/list/new", a.Origin)
	nodes, err := a.chromedp(targetUrl)
	if err != nil {
		return comics, err
	}
	dom, err := goquery.NewDocumentFromReader(strings.NewReader(nodes))
	if err != nil {
		return comics, err
	}
	// 更新列表
	dom.Find(".pure-g > div").Each(func(i int, selection *goquery.Selection) {
		cover, _ := selection.Find("a > amp-img").Attr("src")
		coverSlice := gstr.Split(cover, "?")
		if len(coverSlice) >= 1 {
			cover = coverSlice[0]
		}
		name := selection.Find("a:nth-child(2) > .comics-card__title").Text()
		detailUrl, _ := selection.Find("a:nth-child(2)").Attr("href")
		target := fmt.Sprintf("%v%v", a.Origin, detailUrl)
		comicSet.Add(target)
		comics = append(comics, &model.Comic{
			Name:     name,
			Cover:    cover,
			Origin:   a.Origin,
			Target:   target, //补全完整域名
			State:    0,
			OrderId:  a.OrderId,
			Language: a.Language,
		})
	})
	// 首页列表
	targetUrl = "https://cn.baozimh.com"
	nodes, err = a.chromedp(targetUrl)
	if err != nil {
		return comics, err
	}
	dom, err = goquery.NewDocumentFromReader(strings.NewReader(nodes))
	dom.Find(".l-content").Each(func(i int, selection *goquery.Selection) {
		title := selection.Children().Find(".catalog-title").Text()
		if title == "最近更新" {
			selection.Children().Find(".pure-g > div").Each(func(i int, selection *goquery.Selection) {
				cover, _ := selection.Find("amp-img").Attr("src")
				coverSlice := gstr.Split(cover, "?")
				if len(coverSlice) >= 1 {
					cover = coverSlice[0]
				}
				name := selection.Find(".comics-card__title").Text()
				detailUrl, _ := selection.Find(".comics-card__info").Attr("href")
				target := fmt.Sprintf("%v%v", a.Origin, detailUrl)
				if !comicSet.Contains(target) {
					comics = append(comics, &model.Comic{
						Name:     name,
						Cover:    cover,
						Origin:   a.Origin,
						Target:   target, //补全完整域名
						State:    0,
						OrderId:  a.OrderId,
						Language: a.Language,
					})
				}
			})
		}
	})
	comicSet.Clear()
	return comics, nil
}

// List 漫画列表页面 完成
func (a *baoZi) List(target string) ([]*model.Comic, error) {
	var err error
	comics := make([]*model.Comic, 0)

	nodes, err := a.chromedp(target)
	if err != nil {
		return comics, err
	}

	dom, err := goquery.NewDocumentFromReader(strings.NewReader(nodes))
	if err != nil {
		return comics, err
	}
	dom.Find(".classify-items > div").Each(func(i int, selection *goquery.Selection) {
		cover, _ := selection.Find("amp-img").Attr("src")
		coverSlice := gstr.Split(cover, "?")
		if len(coverSlice) >= 1 {
			cover = coverSlice[0]
		}
		name := selection.Find(".comics-card__title").Text()
		detailUrl, _ := selection.Find(".comics-card__info").Attr("href")
		comics = append(comics, &model.Comic{
			Name:     name,
			Cover:    cover,
			Origin:   a.Origin,
			Target:   fmt.Sprintf("%v%v", a.Origin, detailUrl), //补全完整域名
			State:    0,
			OrderId:  a.OrderId,
			Language: a.Language,
		})
	})
	return comics, err
}

// Details 漫画详情 完成
func (a *baoZi) Details(comic *model.Comic) error {
	var err error
	target := comic.Target
	nodes, err := a.chromedp(target)
	if err != nil {
		return err
	}
	dom, err := goquery.NewDocumentFromReader(strings.NewReader(nodes))
	if err != nil {
		return err
	}
	selection := dom.Find(".comics-detail__info")
	comic.Author = selection.Find(".comics-detail__author").Text()
	selection.Find(".tag-list > span").Each(func(i int, s *goquery.Selection) {
		txt := gstr.Trim(gstr.Replace(s.Text(), "\n", ""))
		if gstr.ContainsI(txt, "完结") {
			comic.State = 2
		}
		if gstr.ContainsI(txt, "连载") {
			comic.State = 1
		}
		if i >= 2 && txt != "" {
			comic.Catalogue += gstr.Trim(txt) + ","
		}
	})
	content := selection.Find(".comics-detail__desc").Text()
	comic.Content = gstr.Trim(gstr.ReplaceByMap(content, map[string]string{}))
	comic.Catalogue = gstr.TrimRight(comic.Catalogue, ",")
	return err
}

// ChapterCount 获取章节总数 完成
func (a *baoZi) ChapterCount(comic *model.Comic) (int, error) {
	count := 0
	var err error
	target := comic.Target
	nodes, err := a.chromedp(target)
	if err != nil {
		return count, err
	}
	dom, err := goquery.NewDocumentFromReader(strings.NewReader(nodes))
	count = dom.Find("#chapter-items > .comics-chapters").Size() + dom.Find("#chapters_other_list > .comics-chapters").Size()
	if count == 0 {
		count = dom.Find(".pure-g > .comics-chapters").Size()
	}
	return count, err
}

// Chapters 获取章节数列表
func (a *baoZi) Chapters(comic *model.Comic) ([]*model.Chapter, error) {
	var err error
	chapters := make([]*model.Chapter, 0)
	target := comic.Target
	nodes, err := a.chromedp(target)
	if err != nil {
		return chapters, err
	}

	dom, err := goquery.NewDocumentFromReader(strings.NewReader(nodes))
	count := dom.Find("#chapter-items > .comics-chapters").Size() + dom.Find("#chapters_other_list > .comics-chapters").Size()

	if count > 0 {
		dom.Find("#chapter-items > .comics-chapters").Each(func(i int, selection *goquery.Selection) {
			name := selection.Find("a").Text()
			targetUrl, _ := selection.Find("a").Attr("href")
			regestr := `chapter_slot=[1-9]\d*`
			reg := regexp.MustCompile(regestr)
			result := reg.FindString(targetUrl)
			sort := gstr.ReplaceByMap(result, map[string]string{
				"chapter_slot=": "",
			})
			chapters = append(chapters, &model.Chapter{
				Name:    gstr.Trim(name),
				Pid:     comic.UUID,
				State:   0,
				Sort:    gconv.Int(sort),
				Target:  fmt.Sprintf("%v%v", a.Origin, targetUrl),
				Origin:  a.Origin,
				OrderId: a.OrderId,
			})
		})
		dom.Find("#chapters_other_list > .comics-chapters").Each(func(i int, selection *goquery.Selection) {
			name := selection.Find("a").Text()
			targetUrl, isExist := selection.Find("a").Attr("href")
			if isExist {
				regStr := `chapter_slot=[1-9]\d*`
				reg := regexp.MustCompile(regStr)
				result := reg.FindString(targetUrl)
				sort := gstr.ReplaceByMap(result, map[string]string{
					"chapter_slot=": "",
				})

				chapters = append(chapters, &model.Chapter{
					Name:    gstr.Trim(name),
					Pid:     comic.UUID,
					State:   0,
					Sort:    gconv.Int(sort),
					Target:  fmt.Sprintf("%v%v", a.Origin, targetUrl),
					Origin:  a.Origin,
					OrderId: a.OrderId,
				})
			}
		})
	} else {
		dom.Find(".pure-g > .comics-chapters").Each(func(i int, selection *goquery.Selection) {
			name := selection.Find("a").Text()
			targetUrl, _ := selection.Find("a").Attr("href")

			regestr := `chapter_slot=[1-9]\d*`
			reg := regexp.MustCompile(regestr)
			result := reg.FindString(targetUrl)
			sort := gstr.ReplaceByMap(result, map[string]string{
				"chapter_slot=": "",
			})

			chapters = append(chapters, &model.Chapter{
				Name:    gstr.Trim(name),
				Pid:     comic.UUID,
				State:   0,
				Sort:    gconv.Int(sort),
				Target:  fmt.Sprintf("%v%v", a.Origin, targetUrl),
				Origin:  a.Origin,
				OrderId: a.OrderId,
			})
		})

	}
	return chapters, err
}

// GetResource 漫画资源 完成
func (a *baoZi) GetResource(targetUrl string) ([]string, error) {
	fmt.Println("开始采集")
	var err error
	resource := make([]string, 0)
	nodes, err := a.chromedp(targetUrl)
	if err != nil {
		return resource, err
	}
	dom, err := goquery.NewDocumentFromReader(strings.NewReader(nodes))
	dom.Find(".comic-contain > div > amp-img").Each(func(i int, selection *goquery.Selection) {
		cover, err1 := selection.Attr("src")
		if err1 == true {
			resource = append(resource, cover)
		}
	})
	return resource, err
}

// 无头浏览器封装
func (a *baoZi) chromedp(target string) (string, error) {
	ctx, cancel := chromedp2.NewBrowser()
	ctx2, cancel2 := chromedp.NewContext(ctx, chromedp.WithLogf(log.Printf))
	defer cancel2()
	ctx, cancel = context.WithTimeout(ctx2, 1*time.Minute)
	defer cancel()
	var nodes string
	err := chromedp.Run(ctx,
		chromedp.Navigate(target),
		chromedp.WaitVisible("#layout", chromedp.ByID),
		chromedp.OuterHTML(`#layout`, &nodes, chromedp.ByID),
	)

	return nodes, err
}
