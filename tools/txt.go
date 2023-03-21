package tools

import (
	"bufio"
	"os"
)

// GetTxtRows 获取文本总行数
func GetTxtRows(file *os.File) int64 {
	var i int64
	input := bufio.NewScanner(file)
	for input.Scan() {
		i++
	}

	return i
}
