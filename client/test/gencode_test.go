package test

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func TestGenCode(t *testing.T) {

	rand.Seed(time.Now().UnixNano())

	fmt.Println(randSeq(10))

}

// 生成10位随机字符串
func randSeq(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	b := make([]rune, n)
	for i := range b {
		b[i] = rune(letters[rand.Intn(len(letters))])
	}
	return string(b)
}
