package utills

import (
	"context"
	"fmt"
	"github.com/axgle/mahonia"
	"github.com/gogf/gf/v2/crypto/gmd5"
	"github.com/gogf/gf/v2/encoding/gbase64"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gfile"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/longbridgeapp/opencc"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"time"
)

func CutStringSlice(slice []string, shareNums int) [][]string {
	sliceLen := len(slice)
	if sliceLen == 0 {
		return nil
	}
	totalShareNums := math.Ceil(float64(sliceLen) / float64(shareNums))
	resSlice := make([][]string, 0, int(totalShareNums))

	for i := 0; i < sliceLen; i += shareNums {
		endIndex := i + shareNums
		if endIndex > sliceLen {
			endIndex = sliceLen
		}
		resSlice = append(resSlice, slice[i:endIndex])
	}
	return resSlice
}

// ToImageBase64 图片转换成base64图片格式
func ToImageBase64(cover, origin string) string {
	ctx := context.TODO()
	client := g.Client()
	client.SetHeader("Referer", origin)
	rsp, err := client.Get(ctx, cover)
	defer rsp.Close()
	if err != nil {
		return cover
	}
	imgByte, _ := ioutil.ReadAll(rsp.Body)
	mimeType := http.DetectContentType(imgByte)
	var baseImg string
	switch mimeType {
	case "image/jpeg":
		baseImg = "data:image/jpeg;base64," + gbase64.EncodeToString(imgByte)
	case "image/png":
		baseImg = "data:image/png;base64," + gbase64.EncodeToString(imgByte)
	default:
		baseImg = fmt.Sprintf("data:%v;base64,%v", mimeType, gbase64.EncodeToString(imgByte))
	}
	return baseImg
}

// ToImageDownLoad 图片下载到本地服务器
func ToImageDownLoad(cover, origin string, orderId int, comicId int64) string {
	ctx := context.TODO()
	client := g.Client()
	client.SetHeader("Referer", origin)
	rsp, err := client.Get(ctx, cover)
	defer rsp.Close()
	if err != nil {
		return cover
	}
	var path = fmt.Sprintf("./upload/cover/%v", orderId)
	if !gfile.Exists(path) {
		if err := gfile.Mkdir(path); err != nil {
			return cover
		}
	}
	var filename = fmt.Sprintf("%v/%v.jpg", path, comicId)
	err = gfile.PutContents(filename, rsp.ReadAllString())
	if err != nil {
		return cover
	}
	return filename
}

func DownloadPicture(url, referer, filename string) error {
	ctx := context.TODO()
	client := g.Client()
	client.SetHeader("Referer", referer)
	rsp, err := client.Get(ctx, url)
	defer rsp.Close()
	if err != nil {
		return err
	}

	err = gfile.PutContents(filename, rsp.ReadAllString())
	if err != nil {
		return err
	}
	return nil
}

/*
*
\ / : * ? " < > |
*/
func ReName(name string) string {
	replaces := map[string]string{
		"//": "",
		"?":  "？",
		",":  "，",
		".":  "。",
		"*":  " ",
		":":  " ",
		"<":  " ",
		">":  " ",
		"\\": "+",
		"|":  " ",
		"\"": "+",
		"	":  " ",
		"\n": " ",
		"/":  "+",
	}
	return gstr.ReplaceByMap(name, replaces)
}

// GetYesterdayDayTimeUnix 获取当天凌晨时间戳
func GetYesterdayDayTimeUnix(day int) int64 {
	t := time.Now()
	addTime := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	timeSamp := addTime.AddDate(0, 0, -day).Unix()
	return timeSamp
}

func GetAfterOneDayTimeUnix() int64 {
	t := time.Now()
	addTime := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	timeSamp := addTime.AddDate(-50, 0, 1).Unix()
	return timeSamp
}

// FtToJt 繁体转换简体
func FtToJt(in string) string {
	t2s, err := opencc.New("t2s")
	if err != nil {
		return in
	}
	out, err := t2s.Convert(in)
	if err != nil {
		return in
	}
	return out
}

func ConvertToString(src string, srcCode string, tagCode string) string {

	srcCoder := mahonia.NewDecoder(srcCode)

	srcResult := srcCoder.ConvertString(src)

	tagCoder := mahonia.NewDecoder(tagCode)

	_, cdata, _ := tagCoder.Translate([]byte(srcResult), true)

	result := string(cdata)

	return result

}

func CheckResource(url, referer string) bool {
	var isPing bool = true
	tryNum := 10
tryOne:
	ctx := context.TODO()
	client := g.Client()
	client.SetHeader("Referer", referer)
	client.SetTimeout(30 * time.Second)
	rsp, err := client.Get(ctx, url)
	defer rsp.Close()
	if err != nil {
		isPing = false
	} else {
		if rsp.StatusCode != 200 || rsp.ContentLength == 0 {
			isPing = false
		}
	}

	if !isPing {
		if tryNum <= 0 {
			return isPing
		}
		tryNum--
		goto tryOne
	}

	return isPing
}

// 免费代理IP
func GetProxy() []string {
	ips := make([]string, 0)
	api := "http://www.66ip.cn/mo.php?sxb=&tqsl=2&port=&export=&ktip=&sxa=&submit=%CC%E1++%C8%A1&textarea=http%3A%2F%2Fwww.66ip.cn%2F%3Fsxb%3D%26tqsl%3D10%26ports%255B%255D2%3D%26ktip%3D%26sxa%3D%26radio%3Dradio%26submit%3D%25CC%25E1%2B%2B%25C8%25A1"
	client := g.Client()
	rsp, err := client.Get(context.TODO(), api)
	if err != nil {
		return ips
	}
	if rsp.StatusCode != 200 {
		return ips
	}
	body := rsp.ReadAllString()
	regStr := "(25[0-5]|2[0-4]\\d|[0-1]\\d{2}|[1-9]?\\d)\\.(25[0-5]|2[0-4]\\d|[0-1]\\d{2}|[1-9]?\\d)\\.(25[0-5]|2[0-4]\\d|[0-1]\\d{2}|[1-9]?\\d)\\.(25[0-5]|2[0-4]\\d|[0-1]\\d{2}|[1-9]?\\d):(\\d)+"
	reg := regexp.MustCompile(regStr)
	ipsTmp := reg.FindAllString(body, -1)
	//测试代理是否可用
	for _, ip := range ipsTmp {
		_, statusCode := ProxyThorn(ip)
		if statusCode == 200 {
			ips = append(ips, fmt.Sprintf("http://%v", ip))
		}
	}
	return ips
}

func ProxyThorn(proxy_addr string) (ip string, status int) {
	//访问查看ip的一个网址
	httpUrl := "https://www.dushimh.com/manhua/nvtudigegexiangshawo/1039256.html"
	proxy, err := url.Parse(proxy_addr)

	netTransport := &http.Transport{
		Proxy:                 http.ProxyURL(proxy),
		MaxIdleConnsPerHost:   10,
		ResponseHeaderTimeout: time.Second * time.Duration(5),
	}
	httpClient := &http.Client{
		Timeout:   time.Second * 10,
		Transport: netTransport,
	}
	res, err := httpClient.Get(httpUrl)
	if err != nil {
		return
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		log.Println(err)
		return
	}
	c, _ := ioutil.ReadAll(res.Body)
	return string(c), res.StatusCode
}

var (
	salt = "hei-comic-wx-#$#-&*^%@#2"
)

// GenerateSign 签名算法
func GenerateSign(data interface{}) string {
	return data.(string)
	return gmd5.MustEncryptString(salt + gmd5.MustEncrypt(data) + gbase64.EncodeString(salt))
}

// CheckSign 校验签名是否正确
func CheckSign(sign string, data interface{}) bool {
	checkSing := GenerateSign(data)
	fmt.Println(sign, checkSing)
	return checkSing == sign
}
