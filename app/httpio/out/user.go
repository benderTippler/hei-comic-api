package out

import "hei-comic-api/app/model"

type User struct {
	Email   string `json:"email"`   //邮箱
	State   int    `json:"state"`   //0 启用  1 禁用
	Sex     int    `json:"sex"`     //0 未知 1 男 2 女
	Device  string `json:"device"`  //设备号
	VipCode string `json:"vipCode"` //邀请码
	Token   string `json:"token"`   //jwt生成的token
}

func (c *User) Marshal(m *model.User) {
	c.Email = m.Email
	c.State = m.State
	c.VipCode = m.VipCode
	c.Sex = m.Sex
}

type ComicUpt struct {
	Count          int64  `json:"count"`
	Source         int    `json:"source"`
	DayUpdateCount int64  `json:"dayUpdateCount"`
	Version        string `json:"version"`
}
