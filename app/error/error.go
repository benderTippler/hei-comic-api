package error

import (
	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
)

var (
	// 999 预留给全局 validate验证器错误信息
	SystemMysqlErr = gerror.NewCode(gcode.New(999, "", nil), "系统错误,请重试！")
	JwtInvalidErr  = gerror.NewCode(gcode.New(998, "", nil), "token 不可以用")
	SignInvalidErr = gerror.NewCode(gcode.New(997, "", nil), "签名校验失败")

	SendEmailCodeErr        = gerror.NewCode(gcode.New(1001, "", nil), "邮箱验证码发送失败，请重试!")
	RegUserCodeErr          = gerror.NewCode(gcode.New(1002, "", nil), "邮箱验证码不匹配")
	RegUserExistErr         = gerror.NewCode(gcode.New(1003, "", nil), "此邮箱已经被注册过了")
	RegUserVipCodeErr       = gerror.NewCode(gcode.New(1004, "", nil), "邀请码不符合规则")
	RegUserVipCodNotFindErr = gerror.NewCode(gcode.New(1005, "", nil), "邀请码不存在")
	RegUserVipCodeUsedErr   = gerror.NewCode(gcode.New(1006, "", nil), "邀请码未被绑定用户,不可用")
	RegUserErr              = gerror.NewCode(gcode.New(1007, "", nil), "注册用户失败，请重试")
	RegUserCodeExpireErr    = gerror.NewCode(gcode.New(1008, "", nil), "邮箱验证码已过期，请重新获取")

	UserNotFindErr       = gerror.NewCode(gcode.New(2001, "", nil), "账号未注册")
	UserPwdErr           = gerror.NewCode(gcode.New(2002, "", nil), "账号与密码不匹配")
	UseNotUsedErr        = gerror.NewCode(gcode.New(2003, "", nil), "账号已经被禁用，请联系管理员！")
	UseQXErr             = gerror.NewCode(gcode.New(2004, "", nil), "账号因取消公众号不再提供服务，请联系管理员！")
	UserCreateTokenErr   = gerror.NewCode(gcode.New(2005, "", nil), "账号token生成失败")
	UserCreateCollectErr = gerror.NewCode(gcode.New(2006, "", nil), "用户收藏失败，请重试")
	UserExistCollectErr  = gerror.NewCode(gcode.New(2007, "", nil), "用户已收藏")
	UserQXCollectErr     = gerror.NewCode(gcode.New(2008, "", nil), "用户取消收藏")
)
