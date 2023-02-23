package adapter

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/dop251/goja"
	"github.com/gocolly/colly"
	"github.com/gogf/gf/v2/container/gset"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
	"hei-comic-api/app/model"
	"hei-comic-api/app/utills"
	"io/ioutil"
	"net/http"
	"regexp"
	"sort"
	"sync"
	"time"
)

var MangaBzAdapter = new(mangaBz)

// 采集章节图片数据
type resourcePage struct {
	resourceSlice []string
	page          int64
}

type mangaBz struct {
	BaseAdapter
	scope []string
}

// 初始化 mangabz 实例对象
func init() {
	err := MangaBzAdapter.InitCfg(6) //初始化配置
	if err != nil {
		panic("MangaBzAdapter InitCfg Fail")
	}
}

// StartUp 采集脚本入口
func (a *mangaBz) StartUp() error {
	return nil
}

func (a *mangaBz) GetUpdateComics() ([]*model.Comic, error) {
	comicSet := gset.New(true)
	comics := make([]*model.Comic, 0)
	page := 1
	for {
		target6Url := fmt.Sprintf("%v/manga-list-0-0-2-p%v/", a.Origin, page)
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
		if page >= 10 {
			break
		}
		page++
	}
	comicSet.Clear()
	return comics, nil
}

// List 漫画列表页面 完毕
func (a *mangaBz) List(target string) ([]*model.Comic, error) {
	comics := make([]*model.Comic, 0)
	cy := colly.NewCollector()
	cy.SetRequestTimeout(10 * time.Minute)
	cy.OnHTML(".container >.mh-list", func(e *colly.HTMLElement) {
		e.ForEach("li", func(i int, item *colly.HTMLElement) {
			name := utills.FtToJt(item.ChildText(".title"))
			cover := item.ChildAttr(".mh-cover", "src")
			detailUrl := fmt.Sprintf("%v%v", a.Origin, item.ChildAttr(".title > a", "href"))
			comics = append(comics, &model.Comic{
				Name:     name,
				Cover:    cover,
				Origin:   a.Origin,
				OrderId:  a.OrderId,
				Language: a.Language,
				Target:   detailUrl,
			})
		})
	})
	err := cy.Visit(target)
	return comics, err
}

// Details 漫画详情 完毕
func (a *mangaBz) Details(comic *model.Comic) error {
	var err error
	cy := colly.NewCollector()
	cy.SetRequestTimeout(10 * time.Minute)
	cy.OnHTML("body", func(e *colly.HTMLElement) {
		author := e.ChildText(".detail-info-tip > span > a")
		stateStr := e.ChildText(".detail-info-tip > span:nth-child(2) > span:nth-child(1)")
		catalogue := make([]string, 0)
		e.DOM.Children().Find(".detail-info-tip > span:nth-child(3) > span").Each(func(i int, selection *goquery.Selection) {
			catalogue = append(catalogue, gstr.ReplaceByMap(gstr.Trim(gstr.Replace(selection.Text(), "題材：", "")), map[string]string{
				"熱血": "热血",
				"戀愛": "恋爱",
				"冒險": "冒险",
				"懸疑": "悬疑",
				"運動": "运动",
				"校園": "校园",
				" ":  "",
			}))
		})
		if gstr.ContainsI(stateStr, "已完結") {
			comic.State = 2
		}
		if gstr.ContainsI(stateStr, "連載中") {
			comic.State = 1
		}

		comic.Author = utills.FtToJt(author)
		if len(catalogue) == 0 {
			catalogue = append(catalogue, "其他")
		}
		comic.Catalogue = gstr.Join(catalogue, ",")

		content := e.ChildText(".detail-info-2>.container>.detail-info>.detail-info-content")
		comic.Content = utills.FtToJt(gstr.Trim(gstr.ReplaceByMap(content, map[string]string{
			"[-折疊]": "",
			"[+展開]": "",
		})))
	})
	err = cy.Visit(comic.Target)
	return err
}

// ChapterCount 获取章节总数 完毕
func (a *mangaBz) ChapterCount(comic *model.Comic) (int, error) {
	count := 0
	cy := colly.NewCollector()
	cy.OnHTML("#chapterlistload", func(e *colly.HTMLElement) {
		a := e.ChildAttrs(".detail-list-form-item", "href")
		count = len(a)
	})
	err := cy.Visit(comic.Target)
	return count, err
}

// Chapters 获取章节数列表 完毕
func (a *mangaBz) Chapters(comic *model.Comic) ([]*model.Chapter, error) {
	chapters := make([]*model.Chapter, 0)
	cy := colly.NewCollector()
	cy.OnHTML("#chapterlistload", func(e *colly.HTMLElement) {
		e.ForEach("a", func(i int, item *colly.HTMLElement) {
			name := gstr.Replace(item.Text, "    ", "")
			target := item.Attr("href")
			regestr := "-?[1-9]\\d*"
			reg := regexp.MustCompile(regestr)
			result := reg.FindString(target)
			sort := gstr.ReplaceByMap(result, map[string]string{})
			chapters = append(chapters, &model.Chapter{
				Name:    gstr.Trim(name),
				Pid:     comic.UUID,
				State:   0,
				Sort:    gconv.Int(sort),
				Target:  fmt.Sprintf("%v%v", a.Origin, target),
				Origin:  a.Origin,
				OrderId: a.OrderId,
			})
		})
	})
	err := cy.Visit(comic.Target)
	return chapters, err
}

// 漫画资源
func (a *mangaBz) GetResource(targetUrl string) ([]string, error) {
	resource := make([]string, 0)
	var js string
	cy := colly.NewCollector()
	cy.OnHTML("head > script:nth-child(11)", func(e *colly.HTMLElement) {
		js = e.Text
	})
	err := cy.Visit(targetUrl)

	if err != nil {
		return resource, err
	}

	js = gstr.ReplaceByMap(js, map[string]string{
		"reseturl(window.location.href, MANGABZ_CURL.substring(0, MANGABZ_CURL.length - 1));": "",
	})

	vm := goja.New()
	_, err = vm.RunString(js)
	if err != nil {
		return resource, err
	}
	domain := vm.Get("MANGABZ_COOKIEDOMAIN").Export().(string)
	curl := vm.Get("MANGABZ_CURL").Export().(string)
	cid := vm.Get("MANGABZ_CID").Export().(int64)
	maxPage := vm.Get("MANGABZ_IMAGE_COUNT").Export().(int64)
	dt := vm.Get("MANGABZ_VIEWSIGN_DT").Export().(string)
	sign := vm.Get("MANGABZ_VIEWSIGN").Export().(string)
	mid := vm.Get("MANGABZ_MID").Export()
	taskImgChan := make(chan bool, 20)
	wg2 := sync.WaitGroup{}
	resourceStruct := make([]*resourcePage, 0, maxPage)
	for page := int64(1); page <= maxPage; page++ {
		wg2.Add(1)
		taskImgChan <- true
		go func(page int64) {
			defer wg2.Done()
		tryOne:
			dierc := fmt.Sprintf("http://%v%vchapterimage.ashx?cid=%v&page=%v&key=&_cid=%v&_mid=%v&_dt=%v&_sign=%v", domain, curl, cid, page, cid, mid, dt, sign)
			method := "GET"
			client := &http.Client{
				Timeout: 10 * time.Minute,
			}
			req, err := http.NewRequest(method, dierc, nil)
			if err != nil {
				fmt.Println("http.NewRequest(method, dierc, nil)", err)
				goto tryOne
			}
			req.Header.Add("Referer", "http://mangabz.com/")
			res, err := client.Do(req)
			if err != nil {
				fmt.Println("client.Do(req)", err)
				goto tryOne
			}
			defer res.Body.Close()
			body, err := ioutil.ReadAll(res.Body)
			if err != nil {
				fmt.Println("ioutil.ReadAll(res.Body)", err)
				goto tryOne
			}
			vm2 := goja.New()
			_, err = vm2.RunString(fmt.Sprintf("var List = %v", gstr.Trim(string(body))))
			if err != nil {
				fmt.Println("m2.RunString", err)
				goto tryOne
			}
			list := vm2.Get("List").Export()
			imageSlice := make([]string, 0)
			for i, v := range list.([]interface{}) {
				if i == 0 {
					imageSlice = append(imageSlice, v.(string))
				}
			}
			resourceStruct = append(resourceStruct, &resourcePage{
				resourceSlice: imageSlice,
				page:          page,
			})
			<-taskImgChan
		}(page)
	}
	wg2.Wait()
	//排序
	sort.SliceStable(resourceStruct, func(i, j int) bool {
		return resourceStruct[i].page < resourceStruct[j].page
	})
	for _, v := range resourceStruct {
		resource = append(resource, v.resourceSlice...)
	}
	return resource, nil
}
