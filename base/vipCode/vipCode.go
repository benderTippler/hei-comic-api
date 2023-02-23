package vipCode

import (
	"fmt"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
	"math"
	"math/big"
	"strconv"
	"strings"
)

// 算法结构: 基础要素 | 校验和 | 位混淆 | 编码转换
//
// 目标码数量: 10亿
// 目标码范围: 12位纯字母全大写 => 目标码可能性 = pow(26, 12) = 95428956661682176 = 95428956亿
// 二进制长度: pow(2, 56) = 72057594037927936 = 72057594亿. 选择56个二进制位做基础数据, 可保证算法结果是12位字母
// 编码转换采用进制转换大法
//
// 礼品码目标量级10亿, 选择32位存储, 支持40亿.
// 24位校验位分配:
//   礼品码类型 8位
//   校验和     4位
//   签名码     12位
//
// 基础数据定义
//   基础数据   := <礼品码类型 8位> <目标码0-7位 8位> <目标码8-15位 8位> <校验和 4位> <目标码16-23位 8位> <签名码 12位> <目标码24-31位 8位>
//   礼品码类型 := 0-127的数字, 表示128种活动. 其中最高位算法保留为1. 确保结果为12位字母.
//   校验和     := 0-15的数字, 根据校验算法, 产生的校验和
//   签名码     := 0-4095的数字, 每种校验算法, 提供固定1个签名码
//   目标码     := 1亿开始顺序增长的数字. 应用层控制

var VipCode = new(vipCode)

const (
	DEBUG         = false
	SIGN          = 2861
	BASE_26_CHARS = "CWEMXSYTQZALIGDVKHNFPJROUB"
)

type vipCode struct {
	sign int64
}

// Generate 生成邀请码
func (v *vipCode) Generate(codeType, target int64) string {
	var code int64
	codeType = codeType & 0x7F //127
	target = target & 0xFFFFFFFF
	sum := v.getSum(target)
	if DEBUG {
		fmt.Println(codeType, sum, target)
	}
	code = (codeType << gconv.Int64(48)) |
		((target & 0xFF000000) << gconv.Int64(16)) |
		((target & 0xFF0000) << gconv.Int64(16)) |
		(sum << gconv.Int64(28)) |
		((target & 0xFF00) << gconv.Int64(12)) |
		(SIGN << gconv.Int64(8)) |
		(target & 0xFF)
	if DEBUG {
		fmt.Println(code)
	}
	// 混淆
	codeStr := v.confuseEncode(code)
	// 补最高位
	codeStr |= 0x80000000000000
	if DEBUG {
		fmt.Println(codeStr)
	}
	return v.convertToNBase(codeStr)
}

// CheckCode 校验验证码
func (v *vipCode) CheckCode(codeStr string) bool {
	code := v.convertFromNBase(codeStr)
	if DEBUG {
		fmt.Println(code)
	}
	// 去最高位
	code &= 0x7FFFFFFFFFFFFF
	if DEBUG {
		fmt.Println(code)
	}
	code = v.confuseDecode(code)
	if DEBUG {
		fmt.Println(code)
	}

	codeType := code >> gconv.Int64(48) & 0x7F
	target0 := (code >> gconv.Int64(40)) & 0xFF
	target1 := (code >> gconv.Int64(32)) & 0xFF
	sum := (code >> gconv.Int64(28)) & 0xF
	target2 := (code >> gconv.Int64(20)) & 0xFF
	sign := (code >> gconv.Int64(8)) & 0xFFF
	target3 := code & 0xFF
	target := (target0 << gconv.Int64(24)) | (target1 << gconv.Int64(16)) | (target2 << gconv.Int64(8)) | target3
	if DEBUG {
		fmt.Println(fmt.Sprintf("type:%v,sum:%v,sign:%v,target:%v", codeType, sum, sign, target))
	}

	if sign != SIGN {
		return false
	}
	return v.checkNum(sign, target, sum)
}

// 校验数目
func (v *vipCode) checkNum(sign, target, sum int64) bool {
	calcSum := v.getSum(target)
	return calcSum == sum
}

/**
 * convertToNBase
 * 十进制转N进制
 * @param string $charTable N进制字母表
 * @param int $number 十进制数字
 * @return string
 */
func (v *vipCode) convertToNBase(code int64) string {
	nbase := gconv.Int64(len(BASE_26_CHARS))
	result := ""
	im := big.NewInt(math.MaxInt64)
	for code > 0 {
		index := im.Mod(big.NewInt(code), big.NewInt(nbase)).Int64()
		result = string(BASE_26_CHARS[index]) + result
		code = im.Div(big.NewInt(code), big.NewInt(nbase)).Int64()
		if DEBUG {
			fmt.Println(fmt.Sprintf("字符：%v,索引：%v,数值：%v", string(BASE_26_CHARS[index]), index, code))
		}
	}
	return result
}

/**
 * convertFromNBase
 * N进制字母表
 * @param string $charTable N进制字母表
 * @param string $number N进制字符串
 * @return int
 */
func (v *vipCode) convertFromNBase(code string) int64 {
	nbase := gconv.Int64(len(BASE_26_CHARS))
	length := gconv.Int64(len(code))
	var (
		offeset int64 = 0
		result  int64 = 0
	)
	for offeset < length {
		chat := string(code[length-offeset-1])
		index := strings.Index(BASE_26_CHARS, chat)
		if index != -1 {
			result += gconv.Int64(math.Pow(gconv.Float64(nbase), gconv.Float64(offeset)) * gconv.Float64(index))
			if DEBUG {
				fmt.Println(fmt.Sprintf("字符：%v,索引：%v,数值：%v", chat, index, result))
			}
			offeset++
		}
	}
	return result
}

// 混淆
func (v *vipCode) confuseEncode(code int64) int64 {
	// 参与混淆的只有55位, 最高位在混淆后补位
	binCode := fmt.Sprintf("%055s", strconv.FormatInt(code, 2)) //十进制转二进制
	if DEBUG {
		fmt.Println(binCode, "长度", len(binCode))
	}
	// 每5位一组， 分11组
	binCodeSlice := make([]string, 0)
	end := gconv.Int(math.Ceil(gconv.Float64(len(binCode)) / 5))
	for i := 0; i < end; i++ {
		binCodeSlice = append(binCodeSlice, gstr.SubStr(binCode, i*5, 5))
	}
	// 整组交换
	if DEBUG {
		fmt.Println("交换之前：", binCodeSlice)
	}
	//交换规则
	rule := map[int]int{
		2: 10,
		1: 5,
		8: 0,
		6: 3,
		4: 9,
	}
	for mk, mv := range rule {
		v.swap(binCodeSlice, mk, mv)
	}
	if DEBUG {
		fmt.Println("交换之后：", binCodeSlice)
		fmt.Println(gstr.Join(binCodeSlice, ""))
	}
	p, _ := strconv.ParseInt(gstr.Join(binCodeSlice, ""), 2, 64) //字符串转换成二进制
	return p
}

// 解密
func (v *vipCode) confuseDecode(code int64) int64 {
	return v.confuseEncode(code)
}

func (v *vipCode) swap(slice []string, a, b int) {
	slice[a], slice[b] = slice[b], slice[a]
}

func (v *vipCode) getSum(target int64) int64 {
	var sum int64 = 0
	tmp := (SIGN << gconv.Int64(32)) | target
	ele0 := tmp & 0x00000000F
	ele1 := (tmp & 0x0000000F0) >> gconv.Int64(4)
	ele2 := (tmp & 0x000000F00) >> gconv.Int64(8)
	ele3 := (tmp & 0x00000F000) >> gconv.Int64(12)
	ele4 := (tmp & 0x0000F0000) >> gconv.Int64(16)
	ele5 := (tmp & 0x000F00000) >> gconv.Int64(20)
	ele6 := (tmp & 0x00F000000) >> gconv.Int64(24)
	ele7 := (tmp & 0x0F0000000) >> gconv.Int64(28)
	ele8 := (tmp & 0xF00000000) >> gconv.Int64(32)
	sum = ele0 + ele1 + ele2 + ele3 + ele4 + ele5 + ele6 + ele7 + ele8
	sum = sum & 0xF
	return sum
}
