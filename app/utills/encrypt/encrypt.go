package encrypt

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
)

var BaseEncrypt = new(baseEncrypt)

type baseEncrypt struct{}

// RsaGenKey 参数bits: 指定生成的秘钥的长度, 单位: bit
func (b *baseEncrypt) RsaGenKey(bits int) error {
	// 1. 生成私钥文件
	// GenerateKey函数使用随机数据生成器random生成一对具有指定字位数的RSA密钥
	// 参数1: Reader是一个全局、共享的密码用强随机数生成器
	// 参数2: 秘钥的位数 - bit
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return err
	}
	// 2. MarshalPKCS1PrivateKey将rsa私钥序列化为ASN.1 PKCS#1 DER编码
	derStream := x509.MarshalPKCS1PrivateKey(privateKey)
	// 3. Block代表PEM编码的结构, 对其进行设置
	block := pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: derStream,
	}
	// 4. 创建文件
	privFile, err := os.Create("./keys/private.pem")
	if err != nil {
		return err
	}
	// 5. 使用pem编码, 并将数据写入文件中
	err = pem.Encode(privFile, &block)
	if err != nil {
		return err
	}
	// 6. 最后的时候关闭文件
	defer privFile.Close()

	// 7. 生成公钥文件
	publicKey := privateKey.PublicKey
	derPkix, err := x509.MarshalPKIXPublicKey(&publicKey)
	if err != nil {
		return err
	}
	block = pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: derPkix,
	}
	pubFile, err := os.Create("./keys/public.pem")
	if err != nil {
		return err
	}
	// 8. 编码公钥, 写入文件
	err = pem.Encode(pubFile, &block)
	if err != nil {
		panic(err)
		return err
	}
	defer pubFile.Close()
	return nil
}

// RSAEncrypt rsa加密
// src 要加密的数据
func (b *baseEncrypt) RSAEncrypt(src []byte) ([]byte, error) {
	result := make([]byte, 0)
	// 1. 根据文件名将文件内容从文件中读出
	filename := "./keys/public.pem"
	file, err := os.Open(filename)
	if err != nil {
		return result, err
	}
	// 2. 读文件
	info, _ := file.Stat()
	allText := make([]byte, info.Size())
	file.Read(allText)
	// 3. 关闭文件
	file.Close()

	// 4. 从数据中查找到下一个PEM格式的块
	block, _ := pem.Decode(allText)
	if block == nil {
		return result, errors.New("加密出错")
	}
	// 5. 解析一个DER编码的公钥
	pubInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return result, err
	}
	pubKey := pubInterface.(*rsa.PublicKey)

	// 6. 公钥加密
	result, err = rsa.EncryptPKCS1v15(rand.Reader, pubKey, src)
	if err != nil {
		return result, err
	}
	return result, nil
}

// RSADecrypt rsa加密
// src 要解密的数据
func (b *baseEncrypt) RSADecrypt(src []byte) []byte {
	// 1. 根据文件名将文件内容从文件中读出
	filename := "./keys/private.pem"
	file, err := os.Open(filename)
	if err != nil {
		return nil
	}
	// 2. 读文件
	info, _ := file.Stat()
	allText := make([]byte, info.Size())
	file.Read(allText)
	// 3. 关闭文件
	file.Close()
	// 4. 从数据中查找到下一个PEM格式的块
	block, _ := pem.Decode(allText)
	// 5. 解析一个pem格式的私钥
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	// 6. 私钥解密
	result, _ := rsa.DecryptPKCS1v15(rand.Reader, privateKey, src)

	return result
}

// =============================================AES加密 开始==============================================

// AESEncrypt aes加密
// src 要加密的数据
// secret 秘钥
func (b *baseEncrypt) AESEncrypt(src, secret []byte) ([]byte, []byte, error) {
	block, err := aes.NewCipher(secret)
	if err != nil {
		return nil, nil, err
	}
	blockSize := block.BlockSize()
	src = PKCS7Padding(src, blockSize)
	iv := secret[:blockSize]
	blockMode := cipher.NewCBCEncrypter(block, iv)
	crypted := make([]byte, len(src))
	blockMode.CryptBlocks(crypted, src)
	return crypted, iv, nil
}

// AESDecrypt 解密数据
// src 要解密的数据
// secret 秘钥
func (b *baseEncrypt) AESDecrypt(src, secret []byte) ([]byte, error) {
	block, err := aes.NewCipher(secret)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	blockMode := cipher.NewCBCDecrypter(block, secret[:blockSize])
	origData := make([]byte, len(src))
	blockMode.CryptBlocks(origData, src)
	origData = PKCS7UnPadding(origData)
	return origData, nil
}

func PKCS7Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

func PKCS7UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}

//=============================================AES加密 结束==============================================
