package goutil

import "math/rand"

func RandStringBytesN(n int) string {
  var letterBytes = []byte("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
  b := make([]byte, n)
  for i := range b {
    b[i] = letterBytes[rand.Int63() % int64(len(letterBytes))]
  }
  return string(b)
}

