package repo

import (
	"errors"
	"fmt"
	"github.com/gogf/gf/v2/crypto/gmd5"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/gogf/gf/v2/util/grand"
	"gorm.io/gorm"
	baseError "hei-comic-api/app/error"
	"hei-comic-api/app/httpio/in"
	"hei-comic-api/app/httpio/out"
	"hei-comic-api/app/middleware"
	"hei-comic-api/app/model"
	"hei-comic-api/app/utills/encrypt"
	"hei-comic-api/base/email"
	"hei-comic-api/base/mysql"
	baseRedis "hei-comic-api/base/redis"
	"hei-comic-api/base/vipCode"
	"math/rand"
	"time"
)

var (
	UserRepo = new(userRepo)
)

type userRepo struct{}

// SendEmailCode  发送邮箱验证码
func (r *userRepo) SendEmailCode(ctx middleware.MyCtx, req *in.SendEmailCode) error {
	redisC := baseRedis.NewRedis()
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	code := fmt.Sprintf("%06v", rnd.Int31n(1000000))
	redisC.Set(ctx.Context, fmt.Sprintf("%v_%v", req.Email, req.Type), code, 60*time.Minute)
	var subject string
	if req.Type == 1 {
		subject = "漫画大师，用户注册码"
	}
	if req.Type == 2 {
		subject = "漫画大师，用户重置密码"
	}
	err := email.NewEmail(ctx.Context).SendEmail(req.Email, subject, fmt.Sprintf("验证码为:%v，10分钟之内有效", code))
	if err != nil {
		return baseError.SendEmailCodeErr
	}
	return nil
}

// Register 注册用户 @TODO 后期加上ip信息，防止异地登录
func (r *userRepo) Register(ctx middleware.MyCtx, req *in.Register) error {
	req.RegVipCode = gstr.ToUpper(req.RegVipCode)
	userM := &model.User{
		Email:   req.RegEmail,
		VipCode: req.RegVipCode,
		Device:  req.RegDevice,
	}
	//第一步、验证码是否有效，验证邀请码是否有效
	redisC := baseRedis.NewRedis()
	code, err := redisC.Get(ctx.Context, fmt.Sprintf("%v_%v", req.RegEmail, 1)).Result()
	if err != nil || code == "" {
		return baseError.RegUserCodeExpireErr
	}

	if code != req.RegCode { //有邮箱收到的验证码不匹配
		return baseError.RegUserCodeErr
	}

	if !vipCode.VipCode.CheckCode(req.RegVipCode) {
		return baseError.RegUserVipCodeErr
	}
	vipCodeResult, err := VipCodeRepo.GetVipCodeByCode(req.RegVipCode)
	if err != nil {
		return err
	}
	userM.VipCode = vipCodeResult.VipCode
	//第二步、验证邮箱验证码是否正确
	userResult, err := r.getUserByEmail(req.RegEmail)
	if err != nil {
		if !errors.Is(err, baseError.UserNotFindErr) {
			return err
		}
	}
	if userResult.UUID > 0 {
		return baseError.RegUserExistErr
	}
	// 第三步、 上面验证都通过
	salt := grand.Letters(8)
	userM.Salt = salt
	userM.Password = encrypt.PasswordEncrypt(salt, req.RegPwd)
	err = r.createUser(userM)
	if err != nil {
		return baseError.RegUserErr
	}
	//删除redis的验证码
	redisC.Del(ctx.Context, fmt.Sprintf("%v_%v", req.RegEmail, 1))
	return nil
}

// 根据邮箱进行查询
func (r *userRepo) getUserByEmail(email string) (*model.User, error) {
	db := mysql.NewDb()
	userM := &model.User{}
	err := db.Model(&model.User{}).Where("email = ?", email).First(&userM).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return userM, baseError.UserNotFindErr
		}
		return userM, baseError.SystemMysqlErr
	}
	return userM, nil
}

// 创建用户
func (r *userRepo) createUser(user *model.User) error {
	db := mysql.NewDb()
	err := db.Model(&model.User{}).Create(user).Error
	if err != nil {
		return err
	}
	return nil
}

// Login 用户登录
func (r *userRepo) Login(ctx middleware.MyCtx, req *in.Login) (*out.User, error) {
	user := &out.User{}
	userM, err := r.getUserByEmail(req.Email)
	if err != nil {
		return user, err
	}
	if userM.State == 1 {
		return user, baseError.UseNotUsedErr
	}
	if userM.State == 2 { //取关公众号
		return user, baseError.UseQXErr
	}

	// 校验密码是否正确
	if !encrypt.CheckPassword(userM.Salt, req.PassWord, userM.Password) {
		return user, baseError.UserPwdErr
	}
	user.Marshal(userM)
	// 生成jwt
	customUserJwt := middleware.CustomUserJwt{
		UUID:     gconv.String(userM.UUID),
		UserType: "user",
	}
	token, err := customUserJwt.GetAccessToken(ctx.Context)
	if err != nil {
		return user, baseError.UserCreateTokenErr
	}
	user.Token = token
	//缓存token到redis,控制设备单一登录
	redisC := baseRedis.NewRedis()
	redisKey := fmt.Sprintf("%v-%v", "login", userM.UUID)
	redisC.Set(ctx.Context, redisKey, gmd5.MustEncryptString(token), 72*time.Hour) //三天有效
	return user, nil
}

func (r *userRepo) UserInfo(ctx middleware.MyCtx, uuid int64) (*model.User, error) {
	db := mysql.NewDb()
	userM := &model.User{}
	err := db.Model(&model.User{}).Select("email,state,phone,sex,vipCode").Where("uuid = ?", uuid).First(&userM).Error
	if err != nil {
		return userM, baseError.SystemMysqlErr
	}
	return userM, nil
}
