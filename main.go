// Kanto
// web service to manage and scale couchdb running on kubernetes
// author: Vaclav Rozsypalek
// Created on 21.06.2016

package main

// imports
import (
	"./kanto"
	"log"
	"net/http"
	"os"
)

// constants
const (
	DEVEL    = true
	LOG_FILE = "./general.log"
)

// MAIN function
func main() {
	// set Log File
	if DEVEL {
		// when developing, do not log into file but straight to stdout
	} else {
		// open logs
		logFile, err := os.Create(LOG_FILE)
		if err != nil {
			kanto.ErrorLog("cannot open log file")
			return
		}
		// set logfile
		log.SetOutput(logFile)
	}
	// init random generator
	kanto.InitRandom()

	// load kubernetes api from os env if configured
	env_kube_api := os.Getenv("KUBERNETES_API_URL")
	if env_kube_api != "" {
		kanto.KUBE_API = env_kube_api
		kanto.InfoLog("kubernetes API url set to: "+env_kube_api)
	} else {
		kanto.InfoLog("kubernetes API url set to default ("+kanto.KUBE_API+"), use env \"KUBERNETES_API_URL\" to set to different value")
	}

	// start kanto web service
	StartWebService()
}

// configure webserver mux and run kanto web service api
// @param none
// @return none
func StartWebService() {
	// configure http path handlers
	mux := kanto.ConfigureWebHandlers()

	// channel for errors
	errChan := make(chan error)
	// prepare Server function for goroutine
	kantoWebService := func(errChan chan error) {
		err := http.ListenAndServe(":80", mux)
		errChan <- err
	}
	// info log
	kanto.InfoLog("starting API service on port 80")

	// start server and listen
	go kantoWebService(errChan)
	// wait for errors
	kanto.ErrorLog(<-errChan)
}
