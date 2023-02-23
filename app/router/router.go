package router

import (
	"github.com/labstack/echo"
	"hei-comic-api/app/controller"
	"hei-comic-api/app/controller/admin"
	"hei-comic-api/app/middleware"
)

var (
	comic  = controller.NewComic()
	user   = controller.NewUser()
	weixin = controller.NewWeiXin()

	dict    = admin.NewDictionary()
	collect = controller.NewCollect()
)

// RegisterRouters app 接口路由
func RegisterRouters(e *echo.Echo) {
	comicGroup := e.Group("/v1")

	comicGroup.POST("/comic/list", comic.List)                      // 漫画列表
	comicGroup.GET("/comic/detail", comic.GetComicById)             // 漫画详情
	comicGroup.GET("/comic/setting", comic.GetComicSetting)         //应用全局配置
	comicGroup.POST("/comic/chapter", comic.GetComicChapters)       //获取漫画的章节列表
	comicGroup.POST("/comic/chapter-top", comic.GetComicChapterTop) //获取漫画的前10章预览图
	comicGroup.GET("/comic/dictionary", comic.FindDict)
	comicGroup.GET("/comic/comicUpt", comic.ComicUpdate)

	comicGroup.POST("/comic/chapter/resource", comic.GetComicChapterResource, middleware.CheckUser) //获取漫画的资源
	comicGroup.POST("/user/send-email", user.SendEmailCode)                                         //发送邮件验证码
	comicGroup.POST("/user/register", user.Register)                                                // 用户注册
	comicGroup.POST("/user/login", user.Login)                                                      // 用户登录
	comicGroup.GET("/user/UserInfo", user.UserInfo)                                                 // 用户信息
	comicGroup.POST("/user/collect", collect.CreateCollect)                                         //用户收藏
	comicGroup.POST("/user/isCollectComic", collect.UserIsCollectComicId)
	comicGroup.GET("/user/UserCollects", collect.UserCollects) //用户收藏列表

	//微信公众号相关接口
	weixinGroup := e.Group("/wx")
	weixinGroup.POST("/user/subscribe", weixin.Subscribe)     //用户关注逻辑
	weixinGroup.POST("/user/unSubscribe", weixin.UnSubscribe) //	用户取关逻辑

}

// RegisterAdminRouters 管理后台接口路由
func RegisterAdminRouters(e *echo.Echo) {
	adminGroup := e.Group("/admin")
	adminGroup.POST("/create/dictionary", dict.CreateDict)
	adminGroup.POST("/comic/comparison", comic.ComparisonComic)
}
