// Kanto
// web service to manage and scale couchdb running on kubernetes
// author: Vaclav Rozsypalek
// Created on 21.06.2016

// log wrapper for easier and more easy-to-read logging

package kanto

import "log"

// constants
const (
	DEBUG = true
)

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

func InfoLog(content interface{}) {
	log.Printf("[INFO]: %v", content)
}
