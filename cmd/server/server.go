package main

import (
	"fmt"
	"github.com/gogf/gf/v2/os/gfile"
	"github.com/labstack/gommon/log"
	"github.com/urfave/cli"
	"hei-comic-api/app/crond"
	"hei-comic-api/base"
	_ "hei-comic-api/base/cache"
	baseMongo "hei-comic-api/base/mongo"
	"hei-comic-api/base/mysql"
	"hei-comic-api/base/redis"
	"hei-comic-api/base/snowflake"
	"hei-comic-api/module/crondServer"
	"hei-comic-api/module/webServer"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

var (
	rootCmd = cli.NewApp()
)

var server = cli.Command{
	Name:      "server",
	ShortName: "s",
	Aliases:   nil,
	Usage:     "server [option]",
	UsageText: "server manage",
	Before:    before,
	Action:    startServer,
}

func before(ctx *cli.Context) error {
	fmt.Println("before--初始化配置")
	// 开始 重写配置文件载入方法
	err := base.InitAdapterCfg()
	if err != nil {
		return err
	}
	// 结束 重写配置文件载入方法

	// 初始化雪花算法节点
	if err = snowflake.InitSnowflake(); err != nil {
		return err
	}

	// 初始化 mongo 数据库
	if err = baseMongo.InitMongo(); err != nil {
		log.Fatal(err)
		return err
	}

	// 初始化 数据库配置
	if err = mysql.InitMysql(); err != nil {
		log.Fatal(err)
		return err
	}

	// 初始化 采集模块数据库表信息
	fmt.Println("before--初始化采集模块数据库表信息")
	if err = base.InitAdapterTable(); err != nil {
		log.Fatal(err)
		return err
	}

	// 初始化 redis 数据库
	fmt.Println("before--初始化redis")
	if err = redis.InitRedis(); err != nil {
		log.Fatal(err)
		return err
	}

	return nil
}

func main() {
	rootCmd.Name = "hei-comic-api"
	rootCmd.UsageText = "app"
	rootCmd.Commands = []cli.Command{
		server,
	}
	if err := rootCmd.Run(os.Args); err != nil {
		log.Error(err)
		os.Exit(1)
	}
}

func startServer(ctx *cli.Context) {

	// web 服务器
	go webServer.Start()

	// 脚本处理 后期移动到脚本服务器上
	go crondServer.Start()

	go func() {
		crond.DownLoadToNas.DownLoadComicsToNas()
	}()

	//定时清除缓存数据
	go func() {
		timer := time.NewTimer(5 * time.Second)
		var temp string
		if runtime.GOOS == "linux" {
			temp = "/tmp/"
		}
		if runtime.GOOS == "windows" {
			temp = "C:\\Users\\Administrator\\AppData\\Local\\Temp"
		}
		for {
			select {
			case <-timer.C:
				fmt.Sprintf("开始清理chromedp采集的缓存数据")
				list, _ := gfile.ScanDir(temp, "chromedp-runner*", false)
				taskChan := make(chan bool, 500)
				for i, v := range list {
					taskChan <- true
					go func(i int, path string) {
						fmt.Println(i)
						gfile.Remove(path)
						<-taskChan
					}(i, v)
				}
			}
			fmt.Sprintf("结束清理chromedp采集的缓存数据")
			timer.Reset(1 * time.Hour)
		}
	}()

	ch := make(chan os.Signal)

	signal.Notify(ch,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGKILL)

	select {
	case <-ch: // 接受到程序终止信号
		//一些关闭操作
		mysql.CloseMysql()
		// 停止执行脚本通知
		crondServer.Stop()
		//通知web 服务器停止服务
		webServer.Stop()
	}

}
