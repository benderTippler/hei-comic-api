package in

type Subscribe struct {
	ToUserName   string `json:"toUserName" validate:"required" label:"开发者微信号"`
	FromUserName string `json:"fromUserName"  validate:"required" label:"发送方帐号"`
	Sign         string `json:"sign"` //签名验证
}

type UnSubscribe struct {
	ToUserName   string `json:"toUserName" validate:"required" label:"开发者微信号"`
	FromUserName string `json:"fromUserName"  validate:"required" label:"发送方帐号"`
	Sign         string `json:"sign"` //签名验证
}
