// Kanto
// web service to manage and scale couchdb running on kubernetes
// author: Vaclav Rozsypalek
// Created on 21.06.2016

// file for kanto struct types
package kanto



/*
 struct that hold all necessary information for couchdb cluster
 */
type CouchdbCluster struct {
	// unique name for cluster
	Tag       string
	// clients username,
	Username  string
	// password to  admin user, generated only at creating time
	Password string `json:",omitempty"`
	// amount fo replicas (pods) in this cluster
	Replicas  int32 `json:",omitempty"`
	// labels used to distinguish an filter this cluster
	Labels    map[string]string `json:"-"`
	// cluster endpoint, which can be used for couchdb http request
	Endpoint  string `json:",omitempty"`
	// kubernetes namespace, where this cluster belongs
	Namespace string `json:",omitempty"`
}

// couchdb struct for couchdb user (database _users)
// example: {"_id":"org.couchdb.user:test","name":"test","type":"user", "roles":["admin"], "password":"test"}
type CouchdbUser struct {
	Id string 	`json:"_id"`
	Name string 	`json:"name"`
	Type string 	`json:"type"`
	Roles []string	`json:"roles"`
	Password string	`json:"password"`
}

// struct for adding durable replication to couchdb
//  '{"_id":"replication__users","source":"_users",
// "target":"http://root:heslo@172.16.20.5:5984/_users",
// "continuous":true,"create_target":true}'

type CouchdbReplicator struct {
	Id string 		`json:"_id"`
	Rev string		`json:"_rev,omitempty"`
	Source string		`json:"source"`
	Target string 		`json:"target"`
	Continuous bool		`json:"continuous"`
	Cancel	bool		`json:"cancel,omitempty"`
}

// replicator record in _replicator DB, we need to query for it before deletion
// {"_id":"replicate_test","_rev":"2-8dce183fb0b0d686161cbf26a50a80a4",
// "source":"http://172.16.20.5:5984/test","target":"http://172.16.20.2:5984/test",
// "continuous":true,"owner":"johny2","_replication_state":"triggered",
// "_replication_state_time":"2016-07-05T08:51:00+00:00","_replication_id":"b97c159dc7dacbf355ef22311823366e"}
type CouchdbReplicatorRecord struct {
	Id string 		`json:"_id"`
	Rev string		`json:"_rev"`
	Source string		`json:"source"`
	Target string 		`json:"target"`
	Continuous bool		`json:"continuous"`
}