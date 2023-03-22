package main

import (
	"bufio"
	"fmt"
	"github.com/Madou-Shinni/file-sync/tools"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
)

const (
	Secret      = "WrEnUzabxW"   // 接口密钥
	FileHashTxt = "fileHash.txt" // hash列表文本
)

var (
	uploadedFiles sync.Map            // 用于记录已上传文件的 MD5 值的 map
	ch            = make(chan string) // 文件读取的消息管道
)

func main() {
	r := gin.Default()

	r.Use(cors.Default()) // 跨域

	r.POST("/file-sync", verifyCode(), uploadHandle)

	// 从管道中读取数据并处理
	for i := 0; i < runtime.NumCPU(); i++ {
		go func() {
			// 从管道中读取需要写入的文本内容
			for line := range ch {
				// 缓存已同步文件hash
				err := writeFileHash(line)
				if err != nil {
					log.Printf("writeFileHash(line):%v", err)
					continue
				}
			}
		}()
	}

	// 创建已同步文件的记录文本
	tools.CreateFile(FileHashTxt)

	// 加载已同步文件缓存
	setUploadedFilesCache(FileHashTxt)

	go r.Run(":8881") // 监听并在 0.0.0.0:8080 上启动服务

	// 监听销毁
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	select {
	case <-signals:
		// 释放资源
		Close()
		fmt.Println("[file-sync] 程序关闭，释放资源")
		return
	}
}

// Close 释放资源
func Close() {
	close(ch)

	// 等待上传goroutine完成
	wg := sync.WaitGroup{}
	wg.Add(runtime.NumCPU())
	for i := 0; i < runtime.NumCPU(); i++ {
		go func() {
			defer wg.Done()
			for range ch {
			}
		}()
	}
	wg.Wait()
}

// 验证code
func verifyCode() gin.HandlerFunc {
	return func(c *gin.Context) {
		code := c.PostForm("code")
		if code != Secret {
			// 验证失败
			c.JSON(403, gin.H{
				"message": "没有权限",
			})
			c.Abort()
			return
		}
	}
}

// 文件上传
func uploadHandle(c *gin.Context) {
	file, err := c.FormFile("file")
	md5sum := c.PostForm("hash")
	dst := c.PostForm("dst")
	if err != nil && dst != "" {
		c.JSON(500, gin.H{
			"message": "参数异常！",
		})
		return
	}

	// 放入文本的内容
	key := file.Filename + "\t" + md5sum

	if _, ok := uploadedFiles.Load(key); ok {
		// 文件已存在
		c.JSON(600, gin.H{
			"message": "文件已存在！",
		})
		return
	}

	// 保存文件
	err = c.SaveUploadedFile(file, dst)
	if err != nil {
		c.JSON(500, gin.H{
			"message": "文件上传失败！",
		})
		return
	}

	// 缓存
	uploadedFiles.Store(key, true)
	// 持久化，将文件hash持久化txt文本
	ch <- key

	c.JSON(http.StatusOK, gin.H{
		"message": "success",
	})
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

// 加载已同步文件hash列表
func setUploadedFilesCache(path string) {
	var (
		err       error
		chunkSize int64
		wg        sync.WaitGroup
	)
	file, err := os.Open(path)
	if err != nil {
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

// 分段读取文本内容
func readFileChunk(file *os.File, start int64, end int64) {
	// 读取文件内容
	_, err := file.Seek(start, 0)
	if err != nil {
		panic(err)
	}
	scanner := bufio.NewScanner(io.NewSectionReader(file, start, end-start))
	for scanner.Scan() {
		// 暂存至内存
		uploadedFiles.Store(scanner, true)
	}
	if err = scanner.Err(); err != nil {
		log.Printf("scanner.Err() err:%v", err)
	}
}
