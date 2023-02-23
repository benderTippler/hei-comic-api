package crond

import (
	"context"
	"fmt"
	"github.com/gogf/gf/v2/frame/g"
	"hei-comic-api/app/collector/adapter"
	"sort"
	"sync"
)

/*
*
把筛选过的漫画下载到本地
*/
var DownLoadToNas = new(downloadToNas)

type downloadToNas struct{}

// DownLoadComicsToNas 下载漫画资源到nas上面
func (c *downloadToNas) DownLoadComicsToNas() error {
	ctx := context.TODO()
	adapterVar, err := g.Cfg().Get(ctx, "adapter")
	if err != nil {
		return err
	}
	adaptersCfg := adapterVar.Vars()
	sort.Slice(adaptersCfg, func(i, j int) bool {
		return adaptersCfg[i].MapStrVar()["sort"].Int() < adaptersCfg[j].MapStrVar()["sort"].Int()
	})
	// 数据资源优先原则，顺序采集
	wg := sync.WaitGroup{}
	taskChan := make(chan bool, 1)
	for _, adapterCfg := range adaptersCfg {
		wg.Add(1)
		taskChan <- true
		go func(adapterCfg *g.Var) {
			defer wg.Done()
			adapterMap := adapterCfg.MapStrVar()
			fmt.Println("开始下载", adapterMap["name"].String())
			isSwitch := adapterMap["switch"].Bool()
			state := adapterMap["state"].Ints()
			if isSwitch {
				adapter := adapter.NewAdapterCollector(adapterMap["orderId"].Int())
				adapter.DownLoadComic(state) //变更更新数据
			}
			fmt.Println("结束下载", adapterMap["name"].String())
			<-taskChan
		}(adapterCfg)
	}
	wg.Wait()
	return nil
}
