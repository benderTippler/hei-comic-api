package webServer

import (
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/tylerb/graceful"
	baseError "hei-comic-api/app/error"
	appmiddleware "hei-comic-api/app/middleware"
	"hei-comic-api/app/router"
	"hei-comic-api/base/validator"
	"log"
	"time"
)

var (
	echoGlobal *echo.Echo
)

func Start() {
	//配置读取
	ctx := gctx.New()
	serverVar, err := g.Cfg().Get(ctx, "server")
	if err != nil {
		log.Fatal(err)
	}
	jwtVar, err := g.Cfg().Get(ctx, "jwt")
	if err != nil {
		log.Fatal(err)
	}
	jwtConf := jwtVar.MapStrVar()

	serverCfg := serverVar.MapStrVar()
	addr := serverCfg["addr"].String()
	autoTls := serverCfg["autoTls"].Bool()
	cert := serverCfg["cert"].String()
	key := serverCfg["key"].String()

	echoGlobal = echo.New()
	echoGlobal.Server.WriteTimeout = 5 * time.Minute
	echoGlobal.Server.ReadTimeout = 5 * time.Minute

	echoGlobal.HideBanner = true
	echoGlobal.Debug = true
	echoGlobal.Validator = validator.Instance()

	//静态文件
	echoGlobal.Static("/comics", "/home/comic/data")

	// 错误处理
	echoGlobal.HTTPErrorHandler = appmiddleware.ErrHandler
	// 全局中间件
	echoGlobal.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
		Skipper:           nil,
		StackSize:         1024,
		DisableStackAll:   false,
		DisablePrintStack: false,
	}))
	// gzip 压缩.
	echoGlobal.Use(middleware.Gzip())
	echoGlobal.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Skipper: nil,
		Format: `{"time":"${time_rfc3339}","id":"${id}","method":"${method}","uri":"${uri}",` +
			`"status":${status},"bytes_in":${bytes_in},"bytes_out":${bytes_out},"remote_ip":"${remote_ip}"}` + "\n",
		CustomTimeFormat: "2006-01-02 15:04:05",
		Output:           nil,
	}))

	//跨越配置
	echoGlobal.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{echo.GET, echo.HEAD, echo.PUT, echo.PATCH, echo.POST, echo.DELETE, echo.OPTIONS},
	}))

	//jwt 功能
	echoGlobal.Use(middleware.JWTWithConfig(middleware.JWTConfig{
		SigningKey:  jwtConf["secret"].Bytes(),
		TokenLookup: "header:" + echo.HeaderAuthorization,
		Skipper: func(c echo.Context) bool {
			// 需要验证jwt的接口
			path := jwtConf["path"].Strings()
			currentPath := c.Request().URL.Path
			currentPaths := gstr.Split(currentPath, "?")
			if len(currentPaths) == 2 {
				currentPath = currentPaths[0]
			}
			if gstr.InArray(path, currentPath) {
				return false
			}
			return true
		},
		ErrorHandler: func(err error) error { //转换成框架错误信息
			return baseError.JwtInvalidErr
		},
		SuccessHandler: func(context echo.Context) { // 解析JWT信息。放到上下文中
			claims := context.Get(middleware.DefaultJWTConfig.ContextKey).(*jwt.Token).Claims.(jwt.MapClaims)
			context.Set("uuid", claims["uuid"])
			context.Set("userType", claims["userType"])
		},
	}))

	router.RegisterRouters(echoGlobal)      // app 接口相关路由
	router.RegisterAdminRouters(echoGlobal) // 后台管理 相关接口

	var errCh = make(chan error, 1)
	if autoTls {
		go func() {
			errCh <- echoGlobal.StartAutoTLS(addr)
		}()
	} else if cert != "" && key != "" {
		errCh <- echoGlobal.StartTLS(addr, cert, key)
	} else {
		go func() {
			errCh <- echoGlobal.Start(addr)
		}()
	}

	for {
		select {
		case err := <-errCh:
			log.Fatal(err)
		}
	}

}

func Stop() {
	graceful.ListenAndServe(echoGlobal.Server, 5*time.Second)
	fmt.Println("web服务器停止")
}
