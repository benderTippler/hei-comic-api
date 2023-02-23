package repo

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
	baseError "hei-comic-api/app/error"
	"hei-comic-api/app/httpio/in"
	"hei-comic-api/app/httpio/out"
	"hei-comic-api/app/middleware"
	"hei-comic-api/app/model"
	"hei-comic-api/base/mysql"
	"hei-comic-api/base/vipCode"
)

var (
	VipCodeRepo = new(vipCodeRepo)
)

type vipCodeRepo struct{}

// CreateVipCodeRepo 一次性创建vipCode
func (r *vipCodeRepo) CreateVipCodeRepo() {
	var codeType int64 = 1 //表示注册邀请码
	var target int64 = 372899
	db := mysql.NewDb()
	chanTask := make(chan bool, 500)
	for {
		target++
		chanTask <- true
		go func(target int64) {
			vipCode := vipCode.VipCode.Generate(codeType, target)
			vipCodeCreate := &model.VipCode{
				VipCode: vipCode,
				Target:  target,
			}
			db.Model(&model.VipCode{}).Create(vipCodeCreate)
			<-chanTask
		}(target)
	}
}

// GetVipCodeByCode 根据vipCode 查询数据
func (r *vipCodeRepo) GetVipCodeByCode(vipCode string) (*model.VipCode, error) {
	db := mysql.NewDb()
	vipCodeM := &model.VipCode{}
	err := db.Model(&model.VipCode{}).Where("vipCode = ?", vipCode).First(&vipCodeM).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return vipCodeM, baseError.RegUserVipCodNotFindErr
		}
		return vipCodeM, baseError.SystemMysqlErr
	}

	if vipCodeM.OpenId == "" { //验证邀请码是否可用
		return vipCodeM, baseError.RegUserVipCodeUsedErr
	}

	return vipCodeM, nil
}

func (r *vipCodeRepo) UpdateVipCodeStateByCode(vipCode string) error {
	db := mysql.NewDb()
	err := db.Model(&model.VipCode{}).Where("vipCode = ?", vipCode).Update("state", 1).Error
	if err != nil {
		return err
	}
	return err
}

// Subscribe 关注微信公众号
func (r *vipCodeRepo) Subscribe(ctx middleware.MyCtx, req *in.Subscribe) (*out.Subscribe, error) {
	db := mysql.NewDb()
	out := &out.Subscribe{}
	//1、 查询当前用户是否曾经取关过。
	vipCode := &model.VipCode{}
	err := db.Model(&model.VipCode{}).Where("openId = ?", req.FromUserName).First(&vipCode).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			out.Tips = "系统错误,请输入 “邀请码” 关键字 重新申请！"
			return out, nil
		}
	}

	if vipCode.UUID > 0 { //账号数据存在，验证用户状态
		user := &model.User{}
		err = db.Model(&model.User{}).Where("vipCode = ?", vipCode.VipCode).First(&user).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) { //邀请码未被用作注册用户
				out.Tips = fmt.Sprintf("您的邀请码：%v，请妥善保存，一个微信号只能获取一个。", vipCode.VipCode)
				out.VipCode = vipCode.VipCode
				return out, nil
			}
			out.Tips = "系统错误,请输入 “邀请码” 关键字 重新申请！"
			return out, nil
		}
		if user.State == 1 { //账号被管理员禁用
			out.Tips = fmt.Sprintf("您的账号：%v，已经被管理员禁用，解除限制请联系管理员，QQ:503186749。", vipCode.VipCode)
			out.VipCode = vipCode.VipCode
			return out, nil
		} else if user.State == 2 { //账号取关了微信公众号
			out.Tips = fmt.Sprintf("因为账号：%v，取关过微信公众号，不再享受此公众号提供的服务，解除限制请联系管理员，QQ:503186749。", user.Email)
			out.VipCode = vipCode.VipCode
			return out, nil
		} else if user.State == 0 { //正常用户
			out.Tips = fmt.Sprintf("您的邀请码：%v，请妥善保存，一个微信号只能获取一个。", vipCode.VipCode)
			out.VipCode = vipCode.VipCode
			return out, nil
		}
	}

	// 第一次关注。随机取邀请码，并且绑定当前用户信息
	err = db.Model(&model.VipCode{}).Where("state = 0").First(&vipCode).Error
	if err != nil {
		out.Tips = "系统错误,请输入 “邀请码” 关键字 重新申请！"
		return out, nil
	}
	// 修改vipCode状态
	err = db.Model(&model.VipCode{}).Where("uuid = ?", vipCode.UUID).Updates(map[string]interface{}{
		"state":  1,                //被使用
		"openId": req.FromUserName, //关注人的数据
	}).Error
	if err != nil {
		out.Tips = "系统错误,请输入 “邀请码” 关键字 重新申请！"
		return out, baseError.SystemMysqlErr
	}
	out.Tips = fmt.Sprintf("您的邀请码：%v，请妥善保存，一个微信号只能获取一个。", vipCode.VipCode)
	out.VipCode = vipCode.VipCode
	return out, nil
}

// UnSubscribe 取注微信公众号
func (r *vipCodeRepo) UnSubscribe(ctx middleware.MyCtx, req *in.UnSubscribe) error {
	db := mysql.NewDb()
	vipCode := &model.VipCode{}
	err := db.Model(&model.VipCode{}).Where("openId = ?", req.FromUserName).First(&vipCode).Error
	if err != nil {
		return nil
	}
	//变更用户状态
	db.Model(&model.User{}).Where("vipCode = ?", vipCode.VipCode).Update("state", 2)
	return nil
}
