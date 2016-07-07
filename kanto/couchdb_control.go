// Kanto
// web service to manage and scale couchdb running on kubernetes
// author: Vaclav Rozsypalek
// Created on 21.06.2016

// file for working with couchdb
package kanto

import (
	"time"

	//
	"k8s.io/kubernetes/pkg/api"
	"github.com/patrickjuchli/couch"
	"errors"
	"strconv"
)

const (
	MAX_RETRIES = 25
	RETRY_WAIT_TIME = 700
	METHOD_POST = "POST"
	METHOD_PUT = "PUT"
	METHOD_GET = "GET"
	METHOD_DELETE = "DELETE"

)

// setup continuous replication between all pods in couchdb cluster
// first it will cancel any replication for all pods
// then it will reinit circle continuous replication between all pods
// requirement -> replicas > 1 !!
// @param cluster - CouchdbCluster struct - cluster where setup replication
func SetupReplication(cluster *CouchdbCluster, databases []string) (error) {
	//DebugLog("Replication setup for all PODS")
	var podList * []api.Pod
	var err error
	for ;; {
		// get pods for this cluster
		podList, err = GetPods(cluster)
		if err != nil {
			ErrorLog("couchdb_control: setup replication - get pods error")
			ErrorLog(err)
			return err
		}
		// check if all pods are already spawned
		if len(*podList) == int(cluster.Replicas) {
			ok := true
			// check if all pods are in state running
			for _, pod := range *podList {
				if pod.Status.Phase != api.PodRunning {
					// pod is not ready yet
					ok = false
					break
				}
			}
			// if we got all pods and all pods are running, then stop waiting and continue with replication
			if ok {
				//DebugLog("All Pods are ready")
				break
			}
		} else {
			// wait for all pods
			time.Sleep(time.Millisecond*RETRY_WAIT_TIME)
		}
	}
	// create couchdb admin credentials
	credentials := couch.NewCredentials(cluster.Username, cluster.Password)

	pods := *podList
	// iterate through all pods
	for i := 0 ; i < len(pods) ; i++ {
		// index of next pod
		j := (i+1) % len(pods)
		DebugLog("couchdb_control: setup replication: pod:"+pods[i].Name+","+pods[i].Status.PodIP )

		// primary - replicate FROM
		server1 := couch.NewServer("http://"+pods[i].Status.PodIP+":"+COUCHDB_PORT_STRING, credentials)
		// check server1
		if err := CheckServer(server1, MAX_RETRIES, RETRY_WAIT_TIME); err != nil {
			// failed to connect to server after all retries, fail replication
			ErrorLog("couchdb_control: setupReplication: failed to connect to server1, pod:"+pods[i].Name)
			ErrorLog(err)
			return err
		}

		// secondary - replicate TO
		server2 := couch.NewServer("http://"+pods[j].Status.PodIP+":"+COUCHDB_PORT_STRING, credentials)
		if err := CheckServer(server2, MAX_RETRIES, RETRY_WAIT_TIME); err != nil {
			// failed to connect to server after all retries, fail replication
			ErrorLog("couchdb_control: setupReplication: failed to connect to server2, pod:"+pods[j].Name)
			ErrorLog(err)
			return err
		}


		// set replication between two pods for all listed databases
		for _, db := range databases {
			// db1 server1
			db1 := server1.Database(db)
			db1.Create()
			// db1  server2
			db2 := server2.Database(db)
			db2.Create()

			// REPLICATION CHOOSE ONLY ONE
			// 1) using _replicate
			// 2) using _replicator

			// replication struct
			replicator := CouchdbReplicator{Id:"replicate_"+db,
						Continuous: true, Source:db1.URL(),
						Target:"http://"+cluster.Username+":"+cluster.Password+"@"+pods[j].Status.PodIP+":"+COUCHDB_PORT_STRING+"/"+db}


			// 1)
			// continuous replication , saves to "_replicate"
			// limits:  anything in _replication is lost when db is restarted

			// DELETE old replication, if any found
			// cannot by done without saving information or without server restart
			// restart server1 and wait until its online, should be fast
			//couch.Do(server1.URL()+"/_restart", METHOD_POST, server1.Cred(), nil, nil)
			//CheckServer(server1, MAX_RETRIES, RETRY_WAIT_TIME)
			// setup replication
			//couch.Do(server1.URL()+"/_replicate", METHOD_POST, server1.Cred(), &replicator, nil)

			// 2)
			// continuous replication via "_replicator"
			// this replication survive restarts but fails replicate database "_users"

			if db == "_users" {
				// there is a bug with _replicator and db _users, we cannot replicate this DB
				continue
			}
			// delete old replication, if any found
			// get old replicator record
			oldReplicator := CouchdbReplicator{}
			couch.Do(server1.URL()+"/_replicator/" + "replicate_" + db , METHOD_GET, server1.Cred(), nil, &oldReplicator)
			//DebugLog(oldReplicator.Rev)
			// if valid replicator record found, delete it
			if oldReplicator.Rev != "" {
				server1.Database("_replicator").Delete("replicate_" + db, oldReplicator.Rev)
			}

			// setup new replication in _replicator db
			couch.Do(server1.URL()+"/_replicator", METHOD_POST, server1.Cred(), &replicator, nil)
		}
	}
	//DebugLog("finished replication configuration")

	// no errors
	return nil
}

// create admin user after couchdb creation
// @param pod - api.Pod - pdo where create admin user
// NOT USED
func CreateAdminUser(server *couch.Server, cluster *CouchdbCluster) (error) {
	/*
	// '{"_id":"org.couchdb.user:test","name":"test","type":"user", "roles":["admin"], "password":"test"}'
	user := CouchdbUser{Id:"org.couchdb.user:"+cluster.Username,
			Password:cluster.Password, Type:"user", Roles:[]string{"admin"}	}
	//
	couch.Do(server.URL()+"_users", "POST", server.Cred(), &user, nil)
	*/
	// no errors
	return nil
}


// check if couchdb server is online, if not it will wait for max retries
// @param server - couch.Server - couchdb server to check
// @param max_retries - int - how many times we should try connect to server
// @param wait_time - int - how long wait for next check (in milisec)
func CheckServer(server *couch.Server, max_retries int, wait_time int) (error) {
	// set max retires
	retries := max_retries
	// test server, "infinite" loop
	for ;; {
		// send request to server
		_ , err := couch.Do(server.URL(), "GET", server.Cred(), nil, nil)
		//server is OK,
		if err == nil  {
			// connection successful, return nil
			return nil
		} else if retries <= 0 {
			// we reached max retry attempts, end with error
			return errors.New("cannot connect to server "+server.URL()+", attempts: "+strconv.Itoa(max_retries))
		} else {
			// server is not responding, try again after a while
			time.Sleep(time.Millisecond * time.Duration(wait_time))
		}
		// reduce retry count
		retries--
	}
}