package crondServer

import (
	"context"
	"fmt"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/robfig/cron/v3"
	"hei-comic-api/app/collector/adapter"
	"hei-comic-api/app/crond"
	baseRedis "hei-comic-api/base/redis"
	"log"
	"sort"
	"time"
)

var (
	crondServer *cron.Cron
	ctx         = context.TODO()
)

// Start 定时脚本服务
func Start() {
	crondServer = cron.New(cron.WithSeconds())
	var errCh = make(chan error, 0)
	var err error
	// 定时任务配置
	redisClient := baseRedis.NewRedis()
	redisClient.Del(ctx, "collector_update_comics") //清除上一次正常退出的key值
	redisClient.Del(ctx, "collector_scan_comics")   //清除上一次正常退出的key值
	redisClient.Del(ctx, "collector_supply_comics") //清除上一次正常退出的key值

	//这个定时任务是采集更新和新增的漫画数据
	_, err = crondServer.AddFunc("0 0/30 * * * ?", func() {
		isLock, err := redisClient.SetNX(ctx, "collector_update_comics", 1, 24*time.Hour).Result()
		if err != nil {
			return
		}
		if isLock {
			fmt.Println("开始 执行脚本")
			adapterVar, err := g.Cfg().Get(ctx, "adapter")
			if err != nil {
				return
			}
			adaptersCfg := adapterVar.Vars()
			sort.Slice(adaptersCfg, func(i, j int) bool {
				return adaptersCfg[i].MapStrVar()["sort"].Int() < adaptersCfg[j].MapStrVar()["sort"].Int()
			})
			// 数据资源优先原则，顺序采集
			for _, adapterCfg := range adaptersCfg {
				adapterMap := adapterCfg.MapStrVar()
				isSwitch := adapterMap["switch"].Bool()
				if isSwitch {
					adapter := adapter.NewAdapterCollector(adapterMap["orderId"].Int())
					adapter.CollectUpdateComics() //变更更新数据
				}
			}
		}
		redisClient.Del(ctx, "collector_update_comics") //清除
		fmt.Println("结束 执行脚本")
	})

	_, err = crondServer.AddFunc("0 0/10 * * * ?", func() {
		isLock, err := redisClient.SetNX(ctx, "collector_scan_comics", 1, 24*time.Hour).Result()
		if err != nil {
			return
		}
		if isLock {
			fmt.Println("开始 漫画的扫描上线 执行脚本")
			crond.OnShelfComicCrond.ScanComics()
			redisClient.Del(ctx, "collector_scan_comics") //清除
			fmt.Println("结束 漫画的扫描上线 执行脚本")
		}
	})

	//_, err = crondServer.AddFunc("0 0 0/2 * * ?", func() {
	//	isLock, err := redisClient.SetNX(ctx, "collector_supply_comics", 1, 24*time.Hour).Result()
	//	if err != nil {
	//		return
	//	}
	//	if isLock {
	//		fmt.Println("开始 漫画的数据补齐操作 执行脚本")
	//		crond.SupplyComicsCrond.SupplyChapters()
	//		redisClient.Del(ctx, "collector_supply_comics") //清除
	//		fmt.Println("结束 漫画的数据补齐操作 执行脚本")
	//	}
	//})

	if err != nil {
		errCh <- err
	}

	go func() {
		crondServer.Start()
	}()

	for {
		select {
		case <-errCh:
			log.Fatal("脚本服务启动失败")
		}
	}
}

func Stop() {
	crondServer.Stop()
	fmt.Println("脚本服务器停止")
}
