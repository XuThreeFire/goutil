package mathutil

import (
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// RandInt [min, max)
func RandInt(min, max int) int { return RandomInt(min, max) }

func RandomInt(min, max int) int {
	rand.Seed(time.Now().UnixNano())
	return min + rand.Intn(max-min) // [0,n)
}
