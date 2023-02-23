package encrypt

import (
	"fmt"
	"github.com/gogf/gf/v2/crypto/gmd5"
	"github.com/gogf/gf/v2/crypto/gsha1"
)

func PasswordEncrypt(salt, password string) string {
	return gmd5.MustEncryptString(fmt.Sprintf("u!2#ser%vhei%vcomic&*", gmd5.MustEncryptString(salt+salt+salt), gsha1.Encrypt(salt+gmd5.MustEncryptString(password))))
}

// CheckPassword 校验输入的密码是否和数据库中密码一致
func CheckPassword(salt, reqPassword, password string) bool {
	return PasswordEncrypt(salt, reqPassword) == password
}
