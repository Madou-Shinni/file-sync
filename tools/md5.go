package tools

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
)

// GetFileMD5 计算文件的 MD5 值
func GetFileMD5(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := md5.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	md5sum := hex.EncodeToString(hasher.Sum(nil))

	return md5sum, nil
}
