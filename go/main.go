package main

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var Version = "dev"

const defaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 Safari/537.36"

type BaiduResponse struct {
	Data struct {
		URL string `json:"url"`
	} `json:"data"`
}

func printHelp() {
	fmt.Println("imgtool", Version)
	fmt.Println("")
	fmt.Println("用法:")
	fmt.Println("  imgtool dataurl-file <本地图片路径>")
	fmt.Println("  imgtool dataurl-url <网络图片地址>")
	fmt.Println("  imgtool upload-file <本地图片路径>")
	fmt.Println("  imgtool upload-url <网络图片地址>")
	fmt.Println("  imgtool version")
	fmt.Println("")
	fmt.Println("说明:")
	fmt.Println("  dataurl-file  本地图片转 data:mimetype;base64,...")
	fmt.Println("  dataurl-url   网络图片转 data:mimetype;base64,...")
	fmt.Println("  upload-file   本地图片上传百度，返回百度图片地址")
	fmt.Println("  upload-url    网络图片上传百度，返回百度图片地址")
	fmt.Println("  version       输出当前版本")
	fmt.Println("")
	fmt.Println("示例:")
	fmt.Println("  ./imgtool dataurl-file ./test.png")
	fmt.Println(`  ./imgtool dataurl-url "https://example.com/a.png"`)
	fmt.Println("  ./imgtool upload-file ./test.png")
	fmt.Println(`  ./imgtool upload-url "https://example.com/a.png"`)
	fmt.Println("  ./imgtool version")
}

func md5Hex(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}

func generateToken(picInfo, timestamp string) string {
	s := md5Hex(picInfo)
	combined := s + "pic_edit" + timestamp
	finalHash := md5Hex(combined)
	if len(finalHash) < 5 {
		return finalHash
	}
	return finalHash[:5]
}

func guessMimeTypeByPath(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if ext != "" {
		if mt := mime.TypeByExtension(ext); mt != "" {
			return mt
		}
	}
	return "application/octet-stream"
}

func normalizeContentType(contentType string) string {
	if contentType == "" {
		return ""
	}
	return strings.TrimSpace(strings.Split(contentType, ";")[0])
}

func fileToBase64(path string) (string, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("文件不存在: %w", err)
	}
	if fi.IsDir() {
		return "", fmt.Errorf("不是文件: %s", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("读取文件失败: %w", err)
	}

	return base64.StdEncoding.EncodeToString(data), nil
}

func fileToDataURL(path string) (string, error) {
	mimeType := guessMimeTypeByPath(path)
	b64, err := fileToBase64(path)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("data:%s;base64,%s", mimeType, b64), nil
}

func urlToDataURL(imageURL string) (string, error) {
	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	req, err := http.NewRequest(http.MethodGet, imageURL, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("User-Agent", defaultUserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("下载网络图片失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("下载网络图片失败: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取网络图片失败: %w", err)
	}

	mimeType := normalizeContentType(resp.Header.Get("Content-Type"))
	if mimeType == "" || !strings.HasPrefix(mimeType, "image/") {
		mimeType = guessMimeTypeByPath(imageURL)
		if mimeType == "application/octet-stream" {
			mimeType = "image/jpeg"
		}
	}

	b64 := base64.StdEncoding.EncodeToString(body)
	return fmt.Sprintf("data:%s;base64,%s", mimeType, b64), nil
}

func buildBaiduRequest(dataURL string) (*http.Request, error) {
	timestamp := fmt.Sprintf("%d", time.Now().UnixMilli())
	token := generateToken(dataURL, timestamp)

	form := url.Values{}
	form.Set("token", token)
	form.Set("scene", "pic_edit")
	form.Set("picInfo", dataURL)
	form.Set("timestamp", timestamp)

	req, err := http.NewRequest(
		http.MethodPost,
		"https://image.baidu.com/aigc/pic_upload",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return nil, fmt.Errorf("创建百度请求失败: %w", err)
	}

	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Origin", "https://image.baidu.com")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Referer", "https://image.baidu.com/")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("User-Agent", defaultUserAgent)
	req.Header.Set("sec-ch-ua", `"Chromium";v="140", "Not=A?Brand";v="24", "Google Chrome";v="140"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"Windows"`)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8")

	return req, nil
}

func uploadDataURLToBaidu(dataURL string) (string, error) {
	req, err := buildBaiduRequest(dataURL)
	if err != nil {
		return "", err
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("上传百度失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取百度响应失败: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("百度返回 HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result BaiduResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("解析百度响应失败: %w, body=%s", err, string(body))
	}

	if result.Data.URL == "" {
		return "", fmt.Errorf("百度返回中未找到图片地址: %s", string(body))
	}

	return result.Data.URL, nil
}

func uploadFileToBaidu(path string) (string, error) {
	dataURL, err := fileToDataURL(path)
	if err != nil {
		return "", err
	}
	return uploadDataURLToBaidu(dataURL)
}

func uploadURLToBaidu(imageURL string) (string, error) {
	dataURL, err := urlToDataURL(imageURL)
	if err != nil {
		return "", err
	}
	return uploadDataURLToBaidu(dataURL)
}

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(1)
	}

	cmd := os.Args[1]

	switch cmd {
	case "-h", "--help", "help":
		printHelp()
		os.Exit(0)
	case "version":
		fmt.Println(Version)
		os.Exit(0)
	}

	if len(os.Args) < 3 {
		fmt.Println("参数不足")
		printHelp()
		os.Exit(1)
	}

	arg := os.Args[2]

	var (
		result string
		err    error
	)

	switch cmd {
	case "dataurl-file":
		result, err = fileToDataURL(arg)
	case "dataurl-url":
		result, err = urlToDataURL(arg)
	case "upload-file":
		result, err = uploadFileToBaidu(arg)
	case "upload-url":
		result, err = uploadURLToBaidu(arg)
	default:
		fmt.Printf("未知命令: %s\n", cmd)
		printHelp()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(2)
	}

	fmt.Println(result)
}
