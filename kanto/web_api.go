// Kanto
// web service to manage and scale couchdb running on kubernetes
// author: Vaclav Rozsypalek
// Created on 21.06.2016

// file for web service API
package kanto

import (
	"net/http"
	"strconv"
	"io"
	"encoding/json"
	"errors"
	"github.com/kubernetes/kubernetes/pkg/api"
	"strings"
)

// struct for json request that are sent to kanto web api
type Request struct {
	credentials User   `json:"credentials"`
	operation   string `json:"operation"`
}

// configure web service api handlers
func ConfigureWebHandlers() *http.ServeMux {
	mux := http.NewServeMux()

	// beta API
	mux.HandleFunc("/v0/database/list", listDatabases)
	mux.HandleFunc("/v0/database/detail", detailDatabase)
	mux.HandleFunc("/v0/database/create", createDatabase)
	mux.HandleFunc("/v0/database/drop", dropDatabase)
	mux.HandleFunc("/v0/database/scale", scaleDatabase)
	mux.HandleFunc("/v0/database/replicate", replicateDatabase)

	// default handler for other requests
	mux.HandleFunc("/", defaultHandler)

	// done
	return mux
}

// http handler
// list all databases clusters that belong to user
func listDatabases(w http.ResponseWriter, r *http.Request) {
	// get user credentials from request
	user := ParseUser(r)
	// check for valid user credentials
	if !user.IsAuthenticated() {
		// sorry
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	// get all couchdb clusters for this user
	clusters := ListCouchdbClusters(user.UserName, api.NamespaceDefault)
	// marshal to json encoded string
	cluster_list, _ := json.Marshal(*clusters)
	// print output
	io.WriteString(w, string(cluster_list))
}

// http handler
// show detail of specified database cluster that belong to user
// ie replicas, hostname etc
func detailDatabase(w http.ResponseWriter, r *http.Request) {
	// get user credentials from request
	user := ParseUser(r)
	// check for valid user credentials
	if !user.IsAuthenticated() {
		// sorry
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	// cluster tag
	cluster_tag := r.FormValue("cluster_tag")
	// labels for cluster components
	labels := make(map[string]string)
	labels[LABEL_USER] = user.UserName
	labels[LABEL_CLUSTER_TAG] = cluster_tag
	// init cluster struct
	couchdb_cluster := &CouchdbCluster{Tag: cluster_tag, Username: user.UserName,
					Namespace: api.NamespaceDefault, Labels: labels}

	deployment, err := GetDeployment(couchdb_cluster)
	if err != nil {
		ErrorLog("kube_control: detailDatabase: failed to get deployment")
		ErrorLog(err)
		// end
		return
	}
	// load active replicas
	couchdb_cluster.Replicas = deployment.Spec.Replicas
	// get service
	service, err := GetService(couchdb_cluster)
	if err != nil {
		ErrorLog("kube_control: detailDatabase: failed to get service")
		ErrorLog(err)
		// end
		return
	}
	// save service endpoint
	couchdb_cluster.Endpoint = "http://" + service.Spec.ClusterIP + ":" + COUCHDB_PORT_STRING

	//we got all info we need, lets print it to user
	cluster_info, _ := json.Marshal(*couchdb_cluster)
	io.WriteString(w, string(cluster_info))
}

// http handler
// create a new database cluster for user
func createDatabase(w http.ResponseWriter, r *http.Request) {
	// get user credentials from request
	user := ParseUser(r)
	// check for valid user credentials
	// dummy check
	if !user.IsAuthenticated() {
		// sorry
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// cluster tag
	cluster_tag := r.FormValue("cluster_tag")
	//  check db tag, if its empty or short  generate a new
	if cluster_tag == ""  || len(cluster_tag) < MIN_CLUSTER_TAG {
		cluster_tag = RandStringName(MAX_CLUSTER_TAG)
	} else  if  len(cluster_tag) > MAX_CLUSTER_TAG {
		// very long db name, trim it
		cluster_tag = cluster_tag[:MAX_CLUSTER_TAG-1]
	}

	// amount of replicas in couchdb cluster
	replicas, _ := strconv.Atoi(r.FormValue("replicas"))
	//safe guard for bad replica numbers (or missing)
	if replicas < 1  {
		replicas = 1
	} else if replicas > MAX_REPLICAS {
		replicas = MAX_REPLICAS
	}

	// labels for cluster components
	labels := make(map[string]string)
	labels[LABEL_USER] = user.UserName
	labels[LABEL_CLUSTER_TAG] = cluster_tag
	// init cluster struct
	couchdb_cluster := &CouchdbCluster{Tag: cluster_tag, Replicas: int32(replicas), Username: user.UserName,
					Namespace: api.NamespaceDefault, Labels: labels, Password: user.Token}

	// create db cluster
	err := CreateCouchdbCluster(couchdb_cluster)
	// check for errors
	if err != nil {
		// fail response
		io.WriteString(w,"cluster db creation failed error:" +err.Error()+"\n")
	} else {
		io.WriteString(w,"cluster db creation successfull, cluster_tag: "+cluster_tag+"\n")
		// print created cluster info
		cluster_info, _ := json.Marshal(*couchdb_cluster)
		io.WriteString(w, string(cluster_info)+"\n")

	}

}

// http handler
// drop specified database cluster
func dropDatabase(w http.ResponseWriter, r *http.Request) {
	// get user credentials from request
	user := ParseUser(r)
	// check for valid user credentials
	if !user.IsAuthenticated() {
		// sorry
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	// get cluster tag
	cluster_tag := r.FormValue("cluster_tag")

	// labels for cluster components
	labels := make(map[string]string)
	labels[LABEL_USER] = user.UserName
	labels[LABEL_CLUSTER_TAG] = cluster_tag
	// init cluster struct
	couchdb_cluster := &CouchdbCluster{Tag: cluster_tag, Username: user.UserName,
					Namespace: api.NamespaceDefault, Labels: labels}

	// check if  cluster tag belong to this user or if its even exist
	var err error
	if deployment, _ := GetDeployment(couchdb_cluster); deployment == nil {
		// no deployment found,  throw an error
		err = errors.New("invalid or non-existing cluster tag")
		return
	} else {
		// delete couchdb cluster
		err = DeleteCouchdbCluster(couchdb_cluster)
	}

	// check for errors
	if err != nil {
		// fail response
		io.WriteString(w,"cluster db deletion failed error:" +err.Error())
	} else {
		io.WriteString(w,"cluster db deletion successfull for cluster_tag: "+cluster_tag)
	}
}

// http handler
// scale up/down specified database cluster
func scaleDatabase(w http.ResponseWriter, r *http.Request) {
	// get user credentials from request
	user := ParseUser(r)
	// check for valid user credentials
	if !user.IsAuthenticated() {
		// sorry
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	// get new amount of replicas
	replicas, _ := strconv.Atoi(r.FormValue("replicas"))
	//safe guard for bad replica numbers (or missing)
	if replicas < 1  {
		replicas = 1
	} else if replicas > MAX_REPLICAS {
		replicas = MAX_REPLICAS
	}

	// cluster tag
	cluster_tag := r.FormValue("cluster_tag")
	// labels for cluster components
	labels := make(map[string]string)
	labels[LABEL_USER] = user.UserName
	labels[LABEL_CLUSTER_TAG] = cluster_tag
	// init cluster struct
	couchdb_cluster := &CouchdbCluster{Tag: cluster_tag, Replicas: int32(replicas), Username: user.UserName,
					Namespace: api.NamespaceDefault, Labels: labels, Password: user.Token}

	// check if  cluster tag belong to this user or if its even exist
	var err error
	if deployment, _ := GetDeployment(couchdb_cluster); deployment == nil {
		// no deployment found,  throw an error
		err = errors.New("invalid or non-existing cluster tag")
		return
	} else {
		// its ok, scale cluster
		err = ScaleCouchdbCluster(couchdb_cluster, deployment)
	}

	// check for errors
	if err != nil {
		// fail response
		io.WriteString(w,"cluster scaling failed error:" +err.Error())
	} else {
		io.WriteString(w,"cluster db scaling successfull, cluster_tag: "+cluster_tag+"\n")
		// print scaled cluster info
		cluster_info, _ := json.Marshal(*couchdb_cluster)
		io.WriteString(w, string(cluster_info)+"\n")
	}


}

// http handler
// edit details of database cluster
func replicateDatabase(w http.ResponseWriter, r *http.Request) {
	// get user credentials from request
	user := ParseUser(r)
	// check for valid user credentials
	if !user.IsAuthenticated() {
		// sorry
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// cluster tag
	cluster_tag := r.FormValue("cluster_tag")
	databases := strings.Split(r.FormValue("databases"), ",")

	// labels for cluster components
	labels := make(map[string]string)
	labels[LABEL_USER] = user.UserName
	labels[LABEL_CLUSTER_TAG] = cluster_tag
	// init cluster struct
	couchdb_cluster := &CouchdbCluster{Tag: cluster_tag, Username: user.UserName,
					Namespace: api.NamespaceDefault, Labels: labels, Password: user.Token}

	deployment, err := GetDeployment(couchdb_cluster)
	if err != nil {
		ErrorLog("web_api - replicate DB : get deployment")
		ErrorLog(err)
		io.WriteString(w, err.Error())
		return
	}

	couchdb_cluster.Replicas = deployment.Spec.Replicas

	// setup replication for specified databases
	err = SetupReplication(couchdb_cluster, databases)

	if err != nil {
		ErrorLog("web_api - replicate DB : setup replication")
		ErrorLog(err)
		io.WriteString(w, "failed to setup replication for dbs \n")
	} else {
		// everything is OK
		io.WriteString(w, "sucesfully configured replication for dbs \n")
	}

}

// default handler for request with bad path
func defaultHandler(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "Welcome to Kanto Web-Service v 0.1 \n" +
			"supported operations: \n" +
			" - create  	/v0/database/create \n" +
			" - drop  	/v0/database/drop \n" +
			" - detail  	/v0/database/detail \n" +
			" - list  	/v0/database/list \n" +
			" - scale  	/v0/database/scale \n" +
			" - replicate  	/v0/database/replicate \n" )

}

