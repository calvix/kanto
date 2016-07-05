package kanto

import (
	"math/rand"
	"time"
)
// init rand seed with current time
func InitRandom() {
    rand.Seed(time.Now().UnixNano())
}
// chars for random string
var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz1234567890")
// generate random string with specified size
func RandStringName(n int) string {
    b := make([]rune, n)
    for i := range b {
        b[i] = letterRunes[rand.Intn(len(letterRunes))]
    }
    return string(b)
}
