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
// @param n - int - length if random string
func RandStringName(n int) string {
    b := make([]rune, n)
    for i := range b {
        b[i] = letterRunes[rand.Intn(len(letterRunes))]
    }
    return string(b)
}

// return couchdb cluster endpoint URL
// @param clusterIp - cluster ip from kubernetes service
func ClusterEndpoint(clusterIp string) (string) {
	return "http://"+clusterIp+":"+COUCHDB_PORT_STRING
}