package in

type SendEmailCode struct {
	Email string `json:"email"`
	Type  int    `json:"type"` //1、注册 2、重置密码
}

// Register 注册用户数据
type Register struct {
	RegEmail   string `json:"regEmail" validate:"required,email" label:"邮箱"`
	RegCode    string `json:"regCode" validate:"required" label:"验证码"`
	RegPwd     string `json:"regPwd" validate:"required,min=6,max=18" label:"密码"`
	RegVipCode string `json:"regVipCode" validate:"required,vipCode" label:"邀请码"`
	RegDevice  string `json:"regDevice" label:"设备号"`
}

// Login 用户登录  @TODO::后续新增 验证码机制，输入错误3次，显示验证码
type Login struct {
	Email    string `json:"email" validate:"required,email" label:"账号"`
	PassWord string `json:"password" validate:"required,min=6,max=18" label:"密码"`
	//Platform string `json:"platform" validate:"required,min=6,max=18" label:"登录平台"`
}
