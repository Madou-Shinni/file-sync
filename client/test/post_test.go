package test

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"testing"
)

func TestPost(t *testing.T) {
	// 创建一个新的 multipart writer
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	// 创建一个 file part
	file, err := os.Open("../config.yml")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	part, err := writer.CreateFormFile("file", "../config.yml")
	if err != nil {
		panic(err)
	}
	_, err = io.Copy(part, file)
	if err != nil {
		panic(err)
	}

	// 添加其他表单字段
	err = writer.WriteField("code", "WrEnUzabxW")
	if err != nil {
		panic(err)
	}

	// 结束 multipart writer
	err = writer.Close()
	if err != nil {
		panic(err)
	}

	// 创建请求
	req, err := http.NewRequest("POST", "http://192.168.110.94:8080/file-sync", body)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	// 检查 HTTP 响应码
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("uploading failed with status code %d", resp.StatusCode)
	}
}
