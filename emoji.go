package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

type Emoji struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	URL      string `json:"url"`
}

type EmojiData struct {
	Emojis []Emoji `json:"emojis"`
}

var mimeExtensions = map[string]string{
	"image/png":        "png",
	"image/jpeg":       "jpg",
	"image/gif":        "gif",
	"image/webp":       "webp",
	"image/svg+xml":    "svg",
	"application/json": "json", // 示例其他类型
}

var illegalChars = []rune{'/', '\\', ':', '*', '?', '"', '<', '>', '|'}

func main() {
	client := createHTTPClient()

	instance := getUserInput("输入实例地址（例如：misskey.io）：", true)
	jsonURL := fmt.Sprintf("https://%s/api/emojis", instance)

	data := fetchEmojiData(client, jsonURL)

	categories := processCategories(data.Emojis)
	selectedCategories := selectCategories(categories)

	downloadDir := setupDownloadDirectory()

	downloadEmojis(client, data.Emojis, selectedCategories, downloadDir)
}

func createHTTPClient() *http.Client {
	fmt.Print("输入网络代理地址（例如：http://127.0.0.1:8080，留空则不使用代理）：")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	proxyURLStr := strings.TrimSpace(scanner.Text())

	if proxyURLStr == "" {
		return &http.Client{}
	}

	proxyURL, err := url.Parse(proxyURLStr)
	if err != nil {
		log.Fatalf("无效的代理地址: %v", err)
	}

	return &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
	}
}

func fetchEmojiData(client *http.Client, url string) *EmojiData {
	resp, err := client.Get(url)
	if err != nil {
		log.Fatal("请求失败:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("API返回错误状态码: %d", resp.StatusCode)
	}

	var data EmojiData
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Fatal("解析JSON失败:", err)
	}
	return &data
}

func processCategories(emojis []Emoji) []string {
	categorySet := make(map[string]bool)
	for _, emj := range emojis {
		catg := emj.Category
		if catg == "" {
			catg = "未分类"
		}
		categorySet[catg] = true
	}

	categories := make([]string, 0, len(categorySet))
	for catg := range categorySet {
		categories = append(categories, catg)
	}
	return categories
}

func selectCategories(categories []string) []string {
	fmt.Println("请选择要下载的分类：")
	for i, catg := range categories {
		fmt.Printf("%d. %s\n", i+1, catg)
	}

	input := getUserInput("输入要下载的分类编号，多个用英文逗号分隔，全部则直接回车：", false)
	if input == "" {
		return categories
	}

	selectedIndices := make(map[int]bool)
	var selected []string
	for _, part := range strings.Split(input, ",") {
		part = strings.TrimSpace(part)
		num, err := strconv.Atoi(part)
		if err != nil {
			log.Fatalf("无效的编号: %q", part)
		}
		if num < 1 || num > len(categories) {
			log.Fatalf("编号 %d 超出有效范围 (1-%d)", num, len(categories))
		}
		if selectedIndices[num-1] {
			continue
		}
		selectedIndices[num-1] = true
		selected = append(selected, categories[num-1])
	}
	return selected
}

func setupDownloadDirectory() string {
	dir := getUserInput("请选择要下载到的目录（默认./myEmojis）：", false)
	if dir == "" {
		dir = "./myEmojis"
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatal("创建目录失败:", err)
	}
	return dir
}

func downloadEmojis(client *http.Client, emojis []Emoji, categories []string, dir string) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, 16) // 并发控制

	for _, emj := range emojis {
		emj := emj
		catg := emj.Category
		if catg == "" {
			catg = "未分类"
		}

		if !contains(categories, catg) {
			continue
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			// 处理文件名
			safeName := sanitizeFilename(emj.Name)
			if safeName == "" {
				log.Printf("无效的文件名: %s", emj.Name)
				return
			}

			// 获取文件扩展名
			ext, err := getFileExtension(client, emj.URL)
			if err != nil {
				log.Printf("获取扩展名失败: %v", err)
				return
			}

			// 创建完整路径
			filePath := filepath.Join(dir, catg, fmt.Sprintf("%s.%s", safeName, ext))
			if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
				log.Printf("创建目录失败: %v", err)
				return
			}

			// 下载文件
			if err := downloadFile(client, emj.URL, filePath); err != nil {
				log.Printf("下载失败: %v", err)
			}
		}()
	}

	wg.Wait()
	fmt.Println("所有下载任务完成")
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func sanitizeFilename(name string) string {
	// 替换非法字符
	for _, c := range illegalChars {
		name = strings.ReplaceAll(name, string(c), "_")
	}
	// 去除首尾空格
	return strings.TrimSpace(name)
}

func getFileExtension(client *http.Client, url string) (string, error) {
	// 发送HEAD请求获取Content-Type
	resp, err := client.Head(url)
	if err != nil {
		return "", fmt.Errorf("HEAD请求失败: %w", err)
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		return getExtFromURL(url), nil
	}

	// 处理包含参数的Content-Type，例如：image/png; charset=utf-8
	contentType = strings.Split(contentType, ";")[0]
	ext, ok := mimeExtensions[strings.ToLower(contentType)]
	if !ok {
		return getExtFromURL(url), nil
	}
	return ext, nil
}

func getExtFromURL(url string) string {
	parts := strings.Split(url, ".")
	if len(parts) < 2 {
		return "dat"
	}
	lastPart := parts[len(parts)-1]
	if strings.Contains(lastPart, "?") {
		lastPart = strings.Split(lastPart, "?")[0]
	}
	return lastPart
}

func downloadFile(client *http.Client, url, path string) error {
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("非200状态码: %d", resp.StatusCode)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, resp.Body); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	fmt.Printf("下载成功: %s\n", path)
	return nil
}

func getUserInput(prompt string, required bool) string {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print(prompt)
		scanner.Scan()
		input := strings.TrimSpace(scanner.Text())
		if !required || input != "" {
			return input
		}
		fmt.Println("输入不能为空，请重新输入！")
	}
}
