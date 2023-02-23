package validator

import (
	"fmt"
	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	zh_translations "github.com/go-playground/validator/v10/translations/zh"
	"github.com/labstack/gommon/log"
	"reflect"
	"regexp"
)

var defaultValidator *Validator

var Trans ut.Translator

type Validator struct {
	validate *validator.Validate
}

// Validate 必须为结构体 或者结构体指针
func (v *Validator) Validate(i interface{}) error {
	if errs := v.validate.Struct(i); errs != nil {
		for _, err := range errs.(validator.ValidationErrors) {
			return err
		}
	}
	return nil
}

func Instance() *Validator {
	if defaultValidator == nil {
		defaultValidator = new(Validator)
		//设置支持语言
		chinese := zh.New()
		english := en.New()
		uni := ut.New(chinese, chinese, english)
		Trans, _ = uni.GetTranslator("zh")
		validate := validator.New()
		//注册自定义函数
		validate.RegisterValidation("vipCode", func(fl validator.FieldLevel) bool {
			// 验证邀请码是否是纯英文
			fmt.Println(fl.Field().String())
			match, _ := regexp.MatchString("^[A-Za-z]+$", fl.Field().String())
			return match
		})
		validate.RegisterTranslation("vipCode", Trans, func(ut ut.Translator) error {
			return ut.Add("vipCode", "邀请码格式错误", false)
		}, func(ut ut.Translator, fe validator.FieldError) string {
			t, _ := ut.T("vipCode", fe.Field())
			return t
		})

		validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := fld.Tag.Get("label")
			return name
		})
		//注册翻译器到校验器
		err := zh_translations.RegisterDefaultTranslations(validate, Trans)
		if err != nil {
			log.Fatal(err)
		}
		defaultValidator.validate = validate
	}
	return defaultValidator
}
