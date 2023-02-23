package adapter

import (
	"hei-comic-api/app/model"
	"math"
	"regexp"
	"strings"
)

// 切片分割
func cutStringSlice(slice []*model.Comic, shareNums int) [][]*model.Comic {
	sliceLen := len(slice)
	if sliceLen == 0 {
		return nil
	}
	totalShareNums := math.Ceil(float64(sliceLen) / float64(shareNums))
	resSlice := make([][]*model.Comic, 0, int(totalShareNums))

	for i := 0; i < sliceLen; i += shareNums {
		endIndex := i + shareNums
		if endIndex > sliceLen {
			endIndex = sliceLen
		}
		resSlice = append(resSlice, slice[i:endIndex])
	}
	return resSlice
}

func trimHtml(src string) string {
	//将HTML标签全转换成小写
	re, _ := regexp.Compile("\\<[\\S\\s]+?\\>")
	src = re.ReplaceAllStringFunc(src, strings.ToLower)
	//去除STYLE
	re, _ = regexp.Compile("\\<style[\\S\\s]+?\\</style\\>")
	src = re.ReplaceAllString(src, "")
	//去除SCRIPT
	re, _ = regexp.Compile("\\<script[\\S\\s]+?\\</script\\>")
	src = re.ReplaceAllString(src, "")
	//去除所有尖括号内的HTML代码，并换成换行符
	re, _ = regexp.Compile("\\<[\\S\\s]+?\\>")
	src = re.ReplaceAllString(src, "\n")
	//去除连续的换行符
	re, _ = regexp.Compile("\\s{2,}")
	src = re.ReplaceAllString(src, "\n")
	return strings.TrimSpace(src)
}
