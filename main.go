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
			kanto.ErrorLog(err)
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
		kanto.InfoLog("ENV: kubernetes API url set to: "+env_kube_api)
	} else {
		kanto.InfoLog("ENV: kubernetes API url set to default ("+kanto.KUBE_API+"), use env \"KUBERNETES_API_URL\" to set to different value")
	}

	// load spawner type
	env_spawner_type := os.Getenv("SPAWNER_TYPE")
	// check env value
	if env_spawner_type == kanto.COMPONENT_RC || env_spawner_type == kanto.COMPONENT_PETSET {
		kanto.SPAWNER_TYPE = env_spawner_type
		kanto.InfoLog("ENV: kanto spawner component set to: \""+env_spawner_type+"\"")
	} else {
		kanto.InfoLog("ENV: kanto spawner component set to default (\""+kanto.SPAWNER_TYPE+"\"), use env \"SPAWNER_TYPE\" to change default spawner. Possible values: rc, deployment (no pv))")
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
