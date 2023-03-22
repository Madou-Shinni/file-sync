package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/Madou-Shinni/file-sync/client/conf"
	_ "github.com/Madou-Shinni/file-sync/client/initialization"
	"github.com/Madou-Shinni/file-sync/tools"
	"github.com/go-co-op/gocron"
	"github.com/urfave/cli/v2"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"
)

const (
	Secret      = "WrEnUzabxW"   // 接口密钥
	FileHashTxt = "fileHash.txt" // hash列表文本
)

var (
	localDir      string   // 本地目录
	uploadedFiles sync.Map // 用于记录已上传文件的 MD5 值的 map
)

func main() {
	localDir = ""
	// 本地上传目录和远程同步目录
	app := &cli.App{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "src",
				Aliases:  []string{"S"},
				Usage:    "需要同步文件的目录",
				Required: true,
			},
		},
		Action: func(c *cli.Context) error {
			localDir = c.String("src")

			// 创建已同步文件的记录文本
			tools.CreateFile(FileHashTxt)
			// 加载已同步文件的hash列表
			setUploadedFilesCache(FileHashTxt)
			// 定时任务
			timezone, _ := time.LoadLocation("Asia/Shanghai")
			s := gocron.NewScheduler(timezone)
			// 每隔一分钟执行一次
			s.Every(1).Minute().Do(timedSynchronization)
			s.StartAsync()

			// 监听销毁
			signals := make(chan os.Signal, 1)
			signal.Notify(signals, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

			select {
			case <-signals:
				// 释放资源
				fmt.Println("[file-sync] 程序关闭，释放资源")
			}

			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

// 定时同步文件
func timedSynchronization() {
	var (
		err              error
		wg               sync.WaitGroup             // 确保函数退出前所有goroutine都结束，避免内存泄漏
		concurrencyLimit = make(chan struct{}, 200) // 限制goroutine数量
	)

	// 获取本地目录中的所有文件
	err = filepath.Walk(localDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 忽略目录
		if info.IsDir() {
			return nil
		}

		wg.Add(1)
		concurrencyLimit <- struct{}{}
		// 并发上传
		go func() {
			defer wg.Done()
			defer func() { <-concurrencyLimit }()

			if err := uploadFiles(path); err != nil {
				log.Printf("failed to upload %s: %v", path, err)
			}
		}()

		return nil
	})

	if err != nil {
		log.Println(err)
	}

	wg.Wait()
}

// 上传文件
func uploadFiles(path string) error {
	// 计算文件的 MD5 值
	md5sum, err := tools.GetFileMD5(path)
	if err != nil {
		return err
	}

	// 取文件名
	key := fmt.Sprintf("%s\t%s", filepath.Base(path), md5sum)

	// 如果文件已经上传过，则跳过
	if _, ok := uploadedFiles.Load(key); ok {
		log.Printf("skipping %s because it has already been uploaded\n", path)
		return nil
	}

	// 打开本地文件
	localFile, err := os.Open(path)
	if err != nil {
		return err
	}
	defer localFile.Close()

	// 设置请求参数
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err != nil {
		return err
	}
	writer.WriteField("code", Secret)
	writer.WriteField("dst", path)
	writer.WriteField("hash", md5sum)
	formFile, err := writer.CreateFormFile("file", path)
	if err != nil {
		return err
	}
	_, err = io.Copy(formFile, localFile)
	if err != nil {
		return err
	}
	err = writer.Close()
	if err != nil {
		return err
	}

	// 创建 HTTP 请求
	req, err := http.NewRequest("POST", conf.Conf.UrlPrefix+"/file-sync", body)
	if err != nil {
		return err
	}

	// 设置请求头 req.Header.Set("Content-Type","multipart/form-data")
	req.Header.Add("Content-Type", writer.FormDataContentType())
	// 执行 HTTP 请求
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 检查 HTTP 响应码
	if resp.StatusCode == 600 {
		// 文件已存在
		log.Printf("skipping %s because it has already been uploaded\n", path)
		return nil
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("uploading %s failed with status code %d", path, resp.StatusCode)
	}

	// 将文件的 MD5 值添加到 map 中，表示已经上传过
	uploadedFiles.Store(key, true)

	// 持久化，将文件hash持久化txt文本
	err = writeFileHash(key)
	if err != nil {
		return err
	}
	// 打印成功信息
	log.Printf("uploaded %s suceess", path)

	return nil
}

// 加载已同步文件hash列表
func setUploadedFilesCache(path string) {
	var (
		err       error
		chunkSize int64
		wg        sync.WaitGroup
	)
	file, err := os.Open(path)
	if err != nil {
		log.Printf("os.Open(path) err:%v", err)
		return
	}

	defer file.Close()

	// 获取文件信息
	fileInfo, err := file.Stat()
	if err != nil {
		log.Printf("file.Stat() err:%v", err)
		return
	}
	// 计算分段大小
	fileSize := fileInfo.Size()
	chunkSize = fileSize / int64(runtime.NumCPU())

	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go func(i int64) {
			// 启动goroutine读取每个文件块的内容
			defer wg.Done()

			// 计算读取的开始位置和结束位置
			start := i * chunkSize
			var end int64
			if i == int64(runtime.NumCPU()-1) {
				end = fileSize
			} else {
				end = start + chunkSize
			}

			// 读取分段内容
			readFileChunk(file, start, end)
		}(int64(i))
	}
	wg.Wait()
}

// 持久化写入文本哈希
func writeFileHash(s string) error {
	fileHashTxt, err := os.OpenFile(FileHashTxt, os.O_WRONLY|os.O_CREATE|os.O_APPEND, os.ModePerm)
	if err != nil {
		return err
	}
	defer fileHashTxt.Close()
	_, err = fileHashTxt.WriteString(s + "\n")
	if err != nil {
		return err
	}
	return nil
}

// 分段读取文本内容
func readFileChunk(file *os.File, start int64, end int64) {
	// 读取文件内容
	_, err := file.Seek(start, 0)
	if err != nil {
		panic(err)
	}
	scanner := bufio.NewScanner(io.NewSectionReader(file, start, end-start))
	for scanner.Scan() {
		// 暂存至内存中
		uploadedFiles.Store(scanner.Text(), true)
	}
	if err = scanner.Err(); err != nil {
		log.Printf("scanner.Err() err:%v", err)
	}
}
