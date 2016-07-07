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

const (
	STATUS_OK = "ok"
	STATUS_ERROR = "error"
	STATUS_UNAUTHORIZED = "unauthorized"
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
	mux.HandleFunc("/v0/list", listDatabases)
	mux.HandleFunc("/v0/detail", detailDatabase)
	mux.HandleFunc("/v0/create", createDatabase)
	mux.HandleFunc("/v0/delete", deleteDatabase)
	mux.HandleFunc("/v0/scale", scaleDatabase)
	mux.HandleFunc("/v0/replicate", replicateDatabase)

	// default handler for other requests
	mux.HandleFunc("/", defaultHandler)

	// done
	return mux
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
		unauthorized(w)
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

	// prepare response
	result := KantoResponse{}

	// check for errors
	if err != nil {
		// fail response
		result.Status = STATUS_ERROR
		result.StatusMessage = "couchdb cluster creation failed"
		result.Error = err.Error()
	} else {
		result.Status = STATUS_OK
		result.StatusMessage = "couchdb cluster creation successfull for cluster_tag: "+cluster_tag
		// print created cluster info
		cluster_info, _ := json.Marshal(*couchdb_cluster)
		result.Result = (*json.RawMessage)(&cluster_info)

	}
	// marshal response to JSON
	result_json, _ := json.Marshal(result)
	// write json result
	io.WriteString(w, string(result_json))
}

// http handler
// drop specified database cluster
func deleteDatabase(w http.ResponseWriter, r *http.Request) {
	// get user credentials from request
	user := ParseUser(r)
	// check for valid user credentials
	if !user.IsAuthenticated() {
		// sorry
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		unauthorized(w)
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

	// prepare response
	result := KantoResponse{}

	// check if  cluster tag belong to this user or if its even exist
	var err error
	if deployment, _ := GetDeployment(couchdb_cluster); deployment == nil {
		// no deployment found,  throw an error
		err = errors.New("invalid or non-existing cluster tag")
		result.Status = STATUS_ERROR
		result.StatusMessage = "couchdb cluster deletion failed error: invalid or non-existing cluster tag"
		result.Error = err.Error()
	} else {
		// delete couchdb cluster
		err = DeleteCouchdbCluster(couchdb_cluster)

		// check for errors
		if err != nil {
			// fail response
			result.Status = STATUS_ERROR
			result.StatusMessage = "couchdb cluster deletion failed"
			result.Error = err.Error()
		} else {
			result.Status = STATUS_OK
			result.StatusMessage = "couchdb cluster deletion successfull for cluster_tag: "+cluster_tag
		}
	}

	// marshal response to JSON
	result_json, _ := json.Marshal(result)
	// write json result
	io.WriteString(w, string(result_json))
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
		unauthorized(w)
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

	// prepare response
	result := KantoResponse{}

	// check if  cluster tag belong to this user or if its even exist
	var err error
	if deployment, _ := GetDeployment(couchdb_cluster); deployment == nil {
		// no deployment found,  throw an error
		err = errors.New("invalid or non-existing cluster tag")
		// fail response
		result.Status = STATUS_ERROR
		result.StatusMessage = "couchdb cluster scaling failed, invalid or non-existing cluster tag"
		result.Error = err.Error()
	} else {
		// its ok, scale cluster
		err = ScaleCouchdbCluster(couchdb_cluster, deployment)

			// check for errors
		if err != nil {
			// fail response
			result.Status = STATUS_ERROR
			result.StatusMessage = "couchdb cluster scaling failed"
			result.Error = err.Error()
		} else {
			result.Status = STATUS_OK
			result.StatusMessage = "couchdb cluster scaling successfull for cluster_tag: "+cluster_tag
			// print scaled cluster info
			cluster_info, _ := json.Marshal(*couchdb_cluster)
			result.Result = (*json.RawMessage)(&cluster_info)
		}
	}

	// marshal response to JSON
	result_json, _ := json.Marshal(result)
	// write json result
	io.WriteString(w, string(result_json))
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
		unauthorized(w)
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

	// prepare response
	result := KantoResponse{}

	deployment, err := GetDeployment(couchdb_cluster)
	if err != nil {
		ErrorLog("web_api - replicate DB : get deployment error")
		ErrorLog(err)
		// fail response
		result.Status = STATUS_ERROR
		result.StatusMessage = "couchdb cluster configure db replication failed, cannot find cluster"
		result.Error = err.Error()
	} else {
		// save replicas number
		couchdb_cluster.Replicas = deployment.Spec.Replicas

		// setup replication for specified databases
		err := SetupReplication(couchdb_cluster, databases)

		if err != nil {
			ErrorLog("web_api - replicate DB : setup replication")
			ErrorLog(err)
			// fail response
			result.Status = STATUS_ERROR
			result.StatusMessage = "couchdb cluster configure db replication failed"
			result.Error = err.Error()
		} else {
			// everything is OK
			result.Status = STATUS_OK
			result.StatusMessage = "couchdb cluster configure db replication successfull for cluster_tag: "+cluster_tag
		}
	}

	// marshal response to JSON
	result_json, _ := json.Marshal(result)
	// write json result
	io.WriteString(w, string(result_json))

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
		unauthorized(w)
		return
	}
	// get all couchdb clusters for this user
	clusters, err := ListCouchdbClusters(user.UserName, api.NamespaceDefault)

	// prepare response
	result := KantoResponse{}

	// check for errors
	if err != nil {
		// fail response
		result.Status = STATUS_ERROR
		result.StatusMessage = "couchdb list clusters failed"
		result.Error = err.Error()
	} else {
		result.Status = STATUS_OK
		result.StatusMessage = "couchdb list clusters successfull for user: "+user.UserName
		// marshal to json encoded string
		cluster_list, _ := json.Marshal(*clusters)
		// save
		result.Result = (*json.RawMessage)(&cluster_list)
	}
	// marshal response to JSON
	result_json, _ := json.Marshal(result)
	// write json result
	io.WriteString(w, string(result_json))
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
		unauthorized(w)
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

	// prepare response
	result := KantoResponse{}

	deployment, err := GetDeployment(couchdb_cluster)
	if err != nil {
		ErrorLog("kube_control: detailDatabase: failed to get deployment")
		ErrorLog(err)
		// end
		// fail response
		result.Status = STATUS_ERROR
		result.StatusMessage = "couchdb cluster detail failed"
		result.Error = err.Error()
	} else {
		// load active replicas
		couchdb_cluster.Replicas = deployment.Spec.Replicas
	}

	// get service
	service, err2 := GetService(couchdb_cluster)
	if err2 != nil {
		ErrorLog("kube_control: detailDatabase: failed to get service")
		ErrorLog(err2)
		// fail response
		result.Status = STATUS_ERROR
		result.StatusMessage = "couchdb cluster detail failed"
		result.Error = err2.Error()
	} else {
		// save service endpoint
		couchdb_cluster.Endpoint = "http://" + service.Spec.ClusterIP + ":" + COUCHDB_PORT_STRING
	}

	// no errors
	if err == nil  && err2 == nil {
		result.Status = STATUS_OK
		result.StatusMessage = "couchdb cluster detail successfull for cluster_tag: "+cluster_tag
		// print created cluster info
		cluster_info, _ := json.Marshal(*couchdb_cluster)
		result.Result = (*json.RawMessage)(&cluster_info)

	}
	// marshal response to JSON
	result_json, _ := json.Marshal(result)
	// write json result
	io.WriteString(w, string(result_json))
}

// default handler for request with bad path
func defaultHandler(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "Welcome to Kanto Web-Service v 0.1 \n" +
			"supported operations: \n" +
			" - create  	/v0/create \n" +
			" - drop  	/v0/delete \n" +
			" - detail  	/v0/detail \n" +
			" - list  	/v0/list \n" +
			" - scale  	/v0/scale \n" +
			" - replicate  	/v0/replicate \n\n"+
			"check README.md for more info about API\n")
}

// return unauthorised response
func unauthorized(w http.ResponseWriter) {
	// prepare response
	result := KantoResponse{Status:STATUS_UNAUTHORIZED, StatusMessage:"Authetication failed"}

	// marshal response to JSON
	result_json, _ := json.Marshal(result)
	// write json result
	io.WriteString(w, string(result_json))
}

