package strutil

import (
	"encoding/base64"
	"math/rand"
	"time"
)

/*
字符串随机函数包
*/

func init() {
	rand.Seed(time.Now().UnixNano())
}

const (
	Number          = "0123456789"                 // len 10
	LowerCaseLetter = "abcdefghijklmnopqrstuvwxyz" // len 26
	HighCaseLetter  = "ABCDEFGHIJKLMNOPQRSTUVWXYZ" // len 26
)

// RandStringNumber 随机纯数字字符串
func RandStringNumber(n int) string {
	return randString(n, Number)
}

// RandStringLowerCase 随机纯小写字母字符串
func RandStringLowerCase(n int) string {
	return randString(n, LowerCaseLetter)
}

// RandStringHighCase 随机纯大写字母字符串
func RandStringHighCase(n int) string {
	return randString(n, HighCaseLetter)
}

// RandStringAllBytes 随机数字+小写+大写字母字符串
func RandStringAllBytes(n int) string {
	return randString(n, Number+LowerCaseLetter+HighCaseLetter)
}

func randString(n int, strSet string) string {
	var (
		bs    = []byte(strSet)
		bsLen = len(bs)
		b     = make([]byte, n)
	)
	for i := 0; i < n; i++ {
		b[i] = bs[rand.Intn(bsLen)] // [0,n)
	}
	return string(b)
}

func RandBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// RandUrlEncodingString The encoding pads the output to a multiple of 4 bytes,
func RandUrlEncodingString(n int) (string, error) {
	b, err := RandBytes(n)
	return base64.URLEncoding.EncodeToString(b), err
}
