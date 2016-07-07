# kanto
web service to create and manage multi instance couchdb databases (with replication) within kubernetes

RECOMMENDED: Read about how couchdb cluster (and replication in cluster) is configured in part #Couchdb Cluster configuration (below API documentation)


API for: 
 * creating couchdb cluster
 * deleting couchdb cluster
 * scaling (more or less replicas in couchdb cluster) 
 * replicate - configure which couchdb databases will be replicated among all replicas
 * listing all couchdb clusters for user
 * show detail about couchdb cluster
 
Check INSTALL.md for instruction how to run/compile kanto

# API DOCUMENTATION
all request to API, should be http POST type, since you always have to provide authentication

auth POST values:
 * **username** - string, required: username to authenticate to kanto service (currently kanto has only dummy auth, so everyone is able to create clusters, but username is still needed)
 * **token** - string, required: auth token for username, it is similar to password

response is in JSON, corresponding struct is defined in kanto/types.go - **KantoResponse**
```go
type KantoResponse struct {
	Status 		    string              `json:"status"`
	StatusMessage 	string              `json:"status_message,omitempty"`
	Error  		    string              `json:"error_detail,omitempty"`
	Result 		    *json.RawMessage    `json:"result,omitempty"`
}
```
response parts:
 * **Status** - request result, only 3 options ("ok","error","unauthorized")
 * **StatusMessage** - more info about what happened
 * **Error** - more info about occurred error, if any
 * **Result** - resulting couchdb cluster info or array of couchdb clusters (struct **CouchdbCluster**)

Error and Result can be omitted depending on operation and operation result.



# create
path:
`/v0/create`

POST values:
 * **cluster_tag** - string,optional; name for new couchdb cluster, if not provided random string is generated, string size 4-12,  bigger string si trimmed, smaller is ignored and treated as empty
 * **replicas**  - int,required; amount of couchdb instances that will be spawned,  has to be number between 1-10, other values will adjusted to fit this range
 * **username** - string, required: username to authenticate to kanto service (currently kanto has only dummy auth, so everyone is able to create clusters, but username is still needed)
 * **token** - string, required: auth token for username, it is similar to password
 
# delete
path:
`/v0/delete`

POST values:
 * **cluster_tag** - string,required; couchdb cluster name/tag that will be deleted
 * **username** - string, required: username to authenticate to kanto service (currently kanto has only dummy auth, so everyone is able to create clusters, but username is still needed)
 * **token** - string, required: auth token for username, it is similar to password
 
# scale
path:
`/v0/scale`

POST values:
 * **cluster_tag** - string,required; couchdb cluster name/tag that will be scaled
 * **replicas**  - int,required; new number for replicas, has to be number between 1-10, other values will adjusted to fit this range
 * **username** - string, required: username to authenticate to kanto service (currently kanto has only dummy auth, so everyone is able to create clusters, but username is still needed)
 * **token** - string, required: auth token for username, it is similar to password
 
# replicate
path:
`/v0/replicate`

POST values:
 * **cluster_tag** - string,required; couchdb cluster name/tag that will be scaled
 * **databases**  - string,required; list of dbs to replicate in couchdb cluster, delimiter is "," example: mydb1,special,test1 
 * **username** - string, required: username to authenticate to kanto service (currently kanto has only dummy auth, so everyone is able to create clusters, but username is still needed)
 * **token** - string, required: auth token for username, it is similar to password

# list
path:
`/v0/list`

POST values:
 * **username** - string, required: username to authenticate to kanto service (currently kanto has only dummy auth, so everyone is able to create clusters, but username is still needed)
 * **token** - string, required: auth token for username, it is similar to password
 
  
# detail
path:
`/v0/detail`

POST values:
 * **cluster_tag** - string,required; couchdb cluster name/tag
 * **username** - string, required: username to authenticate to kanto service (currently kanto has only dummy auth, so everyone is able to create clusters, but username is still needed)
 * **token** - string, required: auth token for username, it is similar to password
 
# api test examples
few examples of using kanto api via **curl**

create couchdb cluster
`curl  127.0.0.1:80/v0/create -d "username=johny2&token=Ty5wvW7LuQ3T&cluster_tag=my-test-db1&replicas=3"`

scale couchdb cluster
`curl  127.0.0.1:80/v0/scale -d "username=johny2&token=Ty5wvW7LuQ3T&cluster_tag=my-test-db1&replicas=5"`

show couchdb cluster detail
`curl  127.0.0.1:80/v0/detail -d "username=johny2&token=Ty5wvW7LuQ3T&cluster_tag=my-test-db1"`

replicate db "mydb" and "special" in cluster
`curl  127.0.0.1:80/v0/replicate -d "username=johny2&token=Ty5wvW7LuQ3T&cluster_tag=my-test-db1&databases=mydb,special"`

list all couchdb clusters for user
`curl  127.0.0.1:80/v0/list -d "username=johny2&token=Ty5wvW7LuQ3T"`

delete couchdb cluster
`curl  127.0.0.1:80/v0/delete -d "username=johny2&token=Ty5wvW7LuQ3T&cluster_tag=my-test-db1"`
 
 
 
# Couchdb Cluster configuration
info about how kanto creates couchdb cluster and how configure replication