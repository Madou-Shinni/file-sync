package tools

import (
	"fmt"
	"os"
)

// CreateFile 文件不存在时创建文件
func CreateFile(path string) {
	var err error
	_, err = os.Lstat(path)
	if os.IsNotExist(err) {
		//创建文件，返回两个值，一是创建的文件，二是错误信息
		file, err := os.Create(path)
		if err != nil { // 如果有错误，打印错误，同时返回
			fmt.Println("err = ", err)
			return
		}
		defer file.Close() // 在退出整个函数时，关闭文件
	}
}

// PathNotExistCreate 目录不存在就创建目录
func PathNotExistCreate(folderPath string) {
	if _, err := os.Stat(folderPath); os.IsNotExist(err) {
		// 文件夹不存在，创建该文件夹
		err = os.MkdirAll(folderPath, 0755)
		if err != nil {
			fmt.Println("创建文件夹失败：", err)
		} else {
			fmt.Println("文件夹创建成功！")
		}
	} else {
		// 文件夹已经存在，不需要进行任何操作
		fmt.Println("文件夹已经存在，无需创建。")
	}
}
