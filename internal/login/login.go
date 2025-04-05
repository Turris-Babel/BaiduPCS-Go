package login

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"image"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
	"github.com/mattn/go-colorable"
	"github.com/mdp/qrterminal"
	"github.com/qjfoidnh/BaiduPCS-Go/internal/pcsconfig"
	"github.com/qjfoidnh/BaiduPCS-Go/requester" // Use requester type
	"github.com/robertkrimen/otto"
)

// --- Structs (copied from main.go) ---

// QrImageData represents the QR code image data response
type QrImageData struct {
	ImageUrl string `json:"imgurl"`
	Errno    int64  `json:"errno"`
	Sign     string `json:"sign"`
}

// QueryData represents the response from querying QR code status
type QueryData struct {
	ChannelV string `json:"channel_v"`
}

// LoginData represents the response after login
type LoginData struct {
	Data SessionData `json:"data"`
}

type SessionData struct {
	Session SessionInfo `json:"session"`
}

type SessionInfo struct {
	Bduss      string `json:"bduss"`
	Stoken     string `json:"stoken"`
	Ptoken     string `json:"ptoken"`
	StokenList string `json:"stokenList"` // 保留原始 JSON 字符串
}

// --- QrCodeLogin Service ---

// QrCodeLogin handles QR code login logic.
type QrCodeLogin struct {
	httpClient *requester.HTTPClient // Changed to requester.HTTPClient
	config     *pcsconfig.PCSConfig  // Added config dependency
}

// NewQrCodeLogin creates a new QrCodeLogin service.
func NewQrCodeLogin(httpClient *requester.HTTPClient, config *pcsconfig.PCSConfig) *QrCodeLogin {
	return &QrCodeLogin{
		httpClient: httpClient,
		config:     config,
	}
}

// GenerateGid 生成随机gid
func (q *QrCodeLogin) GenerateGid() (string, error) {
	vm := otto.New()
	_, err := vm.Run(`
        function generate_gid() {
            return "xxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx".replace(/[xy]/g, function(e) {
                var t = 16 * Math.random() | 0
                var n = "x" === e ? t : 3 & t | 8;
                return n.toString(16)
            }).toUpperCase()
        }
    `)
	if err != nil {
		return "", err
	}
	value, err := vm.Call("generate_gid", nil)
	if err != nil {
		return "", err
	}
	return value.String(), nil
}

// GetQrCode 获取登录二维码链接
func (q *QrCodeLogin) GetQrCode(gid string) (string, string, error) {
	t := int64(time.Now().Unix() * 1000)
	t1 := t + 21232
	t2 := t1 + 4
	callBack := fmt.Sprintf("tangram_guid_%d", t)
	webUrl := fmt.Sprintf("https://passport.baidu.com/v2/api/getqrcode?lp=pc&qrloginfrom=pc&gid=%s&callback=%s&apiver=v3&tt=%d&tpl=netdisk&_=%d",
		gid, callBack, t1, t2)

	headers := map[string]string{
		"Accept":          "*/*",
		"Accept-Encoding": "gzip, deflate, br",
		"Accept-Language": "en,zh-CN;q=0.9,zh;q=0.8",
		"Connection":      "keep-alive",
		"Host":            "passport.baidu.com",
		"Referer":         "https://pan.baidu.com/",
		"Sec-Fetch-Dest":  "script",
		"Sec-Fetch-Mode":  "no-cors",
		"Sec-Fetch-Site":  "same-site",
		"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.116 Safari/537.36",
	}

	// Use injected httpClient
	// Corrected argument order: method, url, post body, headers
	respBody, err := q.httpClient.Fetch(http.MethodGet, webUrl, nil, headers)
	if err != nil {
		return "", "", err
	}

	fmt.Println("Response from getQrCode:", string(respBody))

	parsedBody, err := q.ParseCallBackData(callBack, respBody)
	if err != nil {
		return "", "", err
	}

	var qrData QrImageData
	err = json.Unmarshal(parsedBody, &qrData)
	if err != nil {
		return "", "", err
	}

	if qrData.Errno != 0 {
		return "", "", fmt.Errorf("获取二维码失败, errno: %d", qrData.Errno)
	}

	// 确保 imageUrl 带协议
	if !strings.HasPrefix(qrData.ImageUrl, "http://") && !strings.HasPrefix(qrData.ImageUrl, "https://") {
		qrData.ImageUrl = "https://" + qrData.ImageUrl
	}

	return qrData.ImageUrl, qrData.Sign, nil
}

// DownloadQrCode 下载二维码并在终端上打印生成的二维码图案
func (q *QrCodeLogin) DownloadQrCode(imageUrl string) error {
	if !strings.HasPrefix(imageUrl, "http://") && !strings.HasPrefix(imageUrl, "https://") {
		imageUrl = "https://" + imageUrl
	}

	headers := map[string]string{
		"Accept":          "image/webp,image/apng,image/*,*/*;q=0.8",
		"Accept-Encoding": "gzip, deflate, br",
		"Accept-Language": "en,zh-CN;q=0.9,zh;q=0.8",
		"Connection":      "keep-alive",
		"Host":            "passport.baidu.com",
		"Referer":         "https://pan.baidu.com/",
		"Sec-Fetch-Dest":  "image",
		"Sec-Fetch-Mode":  "no-cors",
		"Sec-Fetch-Site":  "same-site",
		"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.116 Safari/537.36",
	}

	// Use injected httpClient
	// Corrected argument order: method, url, post body, headers
	respBody, err := q.httpClient.Fetch(http.MethodGet, imageUrl, nil, headers)
	if err != nil {
		return fmt.Errorf("下载二维码图片失败: %v", err)
	}

	qrContent, err := q.DecodeQRCode(respBody)
	if err != nil {
		return err
	}

	fmt.Println("扫描以下二维码以登录:")
	q.DisplayQRCode(qrContent)
	return nil
}

// QueryQrCode 查询二维码状态
func (q *QrCodeLogin) QueryQrCode(channelId string, gid string) (string, error) {
	for {
		time.Sleep(2 * time.Second)
		t := time.Now().Unix() * 1000
		callBack := fmt.Sprintf("tangram_guid_%d", t)
		t1 := t + 5
		t2 := t1 + 5

		webUrl := fmt.Sprintf("https://passport.baidu.com/channel/unicast?channel_id=%s&tpl=netdisk&gid=%s&callback=%s&apiver=v3&tt=%d&_=%d",
			channelId, gid, callBack, t1, t2)
		headers := map[string]string{
			"Accept":          "*/*",
			"Accept-Encoding": "gzip, deflate, br",
			"Accept-Language": "en,zh-CN;q=0.9,zh;q=0.8",
			"Connection":      "keep-alive",
			"Host":            "passport.baidu.com",
			"Referer":         "https://pan.baidu.com/",
			"Sec-Fetch-Dest":  "script",
			"Sec-Fetch-Mode":  "no-cors",
			"Sec-Fetch-Site":  "same-site",
			"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.116 Safari/537.36",
		}

		// Use injected httpClient
		// Corrected argument order: method, url, post body, headers
		respBody, err := q.httpClient.Fetch(http.MethodGet, webUrl, nil, headers)
		if err != nil {
			fmt.Println("查询二维码状态错误:", err.Error())
			continue
		}

		parsedBody, err := q.ParseCallBackData(callBack, respBody)
		if err != nil {
			fmt.Println("解析回调数据错误:", err.Error())
			continue
		}

		var queryData QueryData
		err = json.Unmarshal(parsedBody, &queryData)
		if err != nil {
			fmt.Println("JSON 解析错误:", err.Error())
			continue
		}

		channelV := queryData.ChannelV
		var tempMap map[string]interface{}
		err = json.Unmarshal([]byte(channelV), &tempMap)
		if err != nil {
			fmt.Println("解析 channelV 错误:", err.Error())
			continue
		}

		v, ok := tempMap["v"].(string)
		if ok && v != "" {
			return v, nil
		}
	}
}

// Login 触发登录逻辑, 同时写入本地配置 (BDUSS, STOKEN, PTOKEN)
func (q *QrCodeLogin) Login(channelV string) (string, string, string, string, error) {
	t := time.Now().Unix() * 1000
	t1 := t + 225
	callBack := "bd__cbs__ay6xvs"
	webUrl := fmt.Sprintf("https://passport.baidu.com/v3/login/main/qrbdusslogin?v=%d&bduss=%s&loginVersion=v4&qrcode=1&tpl=netdisk&apiver=v3&tt=%d&traceid=&time=%d&alg=v3&callback=%s",
		t, channelV, t, t1, callBack)
	webUrl += "&u=https%253A%252F%252Fpan.baidu.com%252Fdisk%252Fhome"

	headers := map[string]string{
		"Accept":                    "application/json,text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9",
		"Accept-Encoding":           "deflate, br",
		"Accept-Language":           "zh-CN,zh;q=0.9,en;q=0.8,en-GB;q=0.7,en-US;q=0.6",
		"Cache-Control":             "max-age=0",
		"Connection":                "keep-alive",
		"Host":                      "passport.baidu.com",
		"Sec-Fetch-Dest":            "document",
		"Sec-Fetch-Mode":            "navigate",
		"Sec-Fetch-Site":            "none",
		"Sec-Fetch-User":            "?1",
		"Upgrade-Insecure-Requests": "1",
		"User-Agent":                "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.88 Safari/537.36 Edg/87.0.664.66",
	}

	// Use injected httpClient
	// Corrected argument order: method, url, post body, headers
	respBody, err := q.httpClient.Fetch(http.MethodGet, webUrl, nil, headers)
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to fetch URL: %w", err)
	}

	parsedBody, err := q.ParseCallBackData(callBack, respBody)
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to parse callback data: %w", err)
	}

	text := string(parsedBody)

	// 修复 JSON 格式问题并解析 loginData
	text = strings.ReplaceAll(text, "'", "\"") // 将单引号替换为双引号
	text = strings.TrimSpace(text)             // 去除首尾空格
	text = strings.ReplaceAll(text, "\\", "")  // 删除多余的反斜杠

	// 确保 JSON 数据格式有效
	if !json.Valid([]byte(text)) {
		return "", "", "", "", fmt.Errorf("invalid JSON format: %s", text)
	}

	var loginData LoginData
	err = json.Unmarshal([]byte(text), &loginData)
	if err != nil {
		return "", "", "", "", fmt.Errorf("解析 JSON 错误: %w", err)
	}

	// 从 stokenList 中提取 netdisk 的 stoken
	netdiskStoken, err := ExtractNetdiskStoken(loginData.Data.Session.StokenList)
	if err != nil {
		fmt.Println("提取 netdisk 的 stoken 失败:", err)
		return "", "", "", "", err
	}

	// 更新 stoken 为 netdisk 的值
	loginData.Data.Session.Stoken = netdiskStoken

	// 配置 BDUSS, STOKEN, PTOKEN
	bduss := loginData.Data.Session.Bduss
	stoken := loginData.Data.Session.Stoken
	ptoken := loginData.Data.Session.Ptoken
	cookies := fmt.Sprintf("BDUSS=%s;PTOKEN=%s;STOKEN=%s;", bduss, ptoken, stoken)

	// Use the injected config instance
	_, err = q.config.SetupUserByBDUSS(bduss, "", stoken, cookies)
	if err != nil {
		fmt.Println("设置用户失败:", err)
		return "", "", "", "", err
	}
	fmt.Printf("登录成功, BDUSS=%s\nSTOKEN=%s\nPTOKEN=%s\n", bduss, stoken, ptoken)

	return bduss, stoken, ptoken, cookies, nil
}

// GetBdstoken 仅作示例
func (q *QrCodeLogin) GetBdstoken() (string, error) {
	webUrl := "https://tongxunlu.baidu.com"

	// Use injected httpClient
	// Corrected argument order: method, url, post body, headers (headers are nil here)
	respBody, err := q.httpClient.Fetch(http.MethodGet, webUrl, nil, nil)
	if err != nil {
		return "", err
	}

	reg := regexp.MustCompile(`var bdstoken = '(.*?)'`)
	res := reg.FindAllSubmatch(respBody, -1)

	if len(res) > 0 && len(res[0]) > 1 {
		return string(res[0][1]), nil
	}

	return "", errors.New("未找到 bdstoken")
}

// ParseCallBackData 从jsonp形态的回调函数中提取JSON片段
func (q *QrCodeLogin) ParseCallBackData(callBack string, respBody []byte) ([]byte, error) {
	if callBack == "" {
		return nil, errors.New("回调函数名为空")
	}
	reg := regexp.MustCompile(fmt.Sprintf(`%s\(([\s\S]+?)\)`, regexp.QuoteMeta(callBack)))
	res := reg.FindAllSubmatch(respBody, -1)

	if len(res) <= 0 {
		fmt.Println("回调数据:", string(respBody))
		return nil, errors.New("未找到匹配的回调数据")
	}

	fmt.Println("Parsed callback data:", string(res[0][1]))
	return res[0][1], nil
}

// DecodeQRCode 将二维码图片的二进制数据解码解析出二维码内容
func (q *QrCodeLogin) DecodeQRCode(data []byte) (string, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("failed to decode image: %v", err)
	}

	bmp, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		return "", fmt.Errorf("failed to create binary bitmap: %v", err)
	}

	qrReader := qrcode.NewQRCodeReader()
	result, err := qrReader.Decode(bmp, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decode QR code: %v", err)
	}
	return result.GetText(), nil
}

// DisplayQRCode 控制台生成二维码图案
func (q *QrCodeLogin) DisplayQRCode(content string) {
	config := qrterminal.Config{
		Level:     qrterminal.L,
		Writer:    colorable.NewColorableStdout(),
		BlackChar: qrterminal.BLACK,
		WhiteChar: qrterminal.WHITE,
		QuietZone: 1,
	}
	qrterminal.GenerateWithConfig(content, config)
}

// --- Helper functions (copied from main.go) ---

// htmlUnescape 解码 HTML 实体 (using standard library)
func htmlUnescape(input string) string {
	return html.UnescapeString(input)
}

// ExtractNetdiskStoken 从 stokenList 中提取 netdisk 的 stoken
func ExtractNetdiskStoken(stokenList string) (string, error) {
	decodedList := htmlUnescape(stokenList) // 解码 HTML 实体
	stokens := strings.Split(decodedList, ",")
	for _, item := range stokens {
		// Correctly escape the double quote in the prefix check
		if strings.HasPrefix(item, "\"netdisk#") { // 查找 netdisk 前缀
			parts := strings.Split(item, "#")
			if len(parts) == 2 {
				return strings.Trim(parts[1], `"`), nil
			}
		}
	}
	return "", errors.New("未找到 netdisk 的 stoken")
}
