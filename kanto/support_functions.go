// Kanto
// web service to manage and scale couchdb running on kubernetes
// author: Vaclav Rozsypalek
// Created on 21.06.2016

// file fr support function
package kanto

import (
	"math/rand"
	"time"
	"log"
)

// constants
const (
	DEBUG = true
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



// debug log
func DebugLog(content interface{}) {
	if DEBUG {
		log.Printf("[debug]: %v", content)
	}
}
// error log
func ErrorLog(content interface{}) {
	log.Printf("[ERROR]: %v", content)
}
// info log
func InfoLog(content interface{}) {
	log.Printf("[INFO]: %v", content)
}
