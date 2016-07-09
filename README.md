# kanto
kanto is a service that can create and manage multi instance couchdb databases (with continuous replication) in kubernetes

REQUIRES: kubernetes v 1.2.1+ 

WARNING: this is not ready-to-production service, since its lacking authentication for users and it is completely stateless.
Check [Limitations](#limitations) section for more info about what is missing.

RECOMMENDED: Read about how couchdb cluster (and replication in cluster) is configured in part #Couchdb Cluster configuration (below API documentation)
 
Check [INSTALL.md](https://github.com/calvix/kanto/blob/master/INSTALL.md) for instruction how to run/compile kanto

# ENVS
kanto uses enviroment values to fetch some configuration values
env list:
 * **KUBERNETES_API_URL** - url to kubernetes api server (defaults to 127.0.0.1:8080)
 * **SPAWNER_TYPE** - decide what kind of component will spawn pods in kuberentes (possible values: "deployment" (default, np pv), "rc")

check [kubernetes info](#kubernetes-info)  more information about SPAWNER_TYPE


# API DOCUMENTATION
API operations:
 * creating couchdb cluster
 * deleting couchdb cluster
 * scaling (more or less replicas in couchdb cluster) 
 * replicate - configure which couchdb databases will be replicated among all replicas
 * listing all couchdb clusters for user
 * show detail about couchdb cluster

all request to API, should be http POST, since you always have to provide authentication (username + token)

auth POST values:
 * **username** - string, required: username to authenticate to kanto service (currently kanto has only dummy auth, so everyone is able to create clusters, but username is still needed)
 * **token** - string, required: auth token for username, it is similar to password

API response is in JSON, corresponding struct is defined in kanto/types.go - **KantoResponse**
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



##create
path:
`/v0/create`

POST values:
 * **cluster_tag** - string,optional; name for new couchdb cluster, if not provided random string is generated, string size 4-12,  bigger string si trimmed, smaller is ignored and treated as empty
 * **replicas**  - int,required; amount of couchdb instances that will be spawned,  has to be number between 1-10, other values will adjusted to fit this range
 * **username** - string, required: username to authenticate to kanto service (currently kanto has only dummy auth, so everyone is able to create clusters, but username is still needed)
 * **token** - string, required: auth token for username, it is similar to password
 
##delete
path:
`/v0/delete`

POST values:
 * **cluster_tag** - string,required; couchdb cluster name/tag that will be deleted
 * **username** - string, required: username to authenticate to kanto service (currently kanto has only dummy auth, so everyone is able to create clusters, but username is still needed)
 * **token** - string, required: auth token for username, it is similar to password
 
##scale
path:
`/v0/scale`

POST values:
 * **cluster_tag** - string,required; couchdb cluster name/tag that will be scaled
 * **replicas**  - int,required; new number for replicas, has to be number between 1-10, other values will adjusted to fit this range
 * **username** - string, required: username to authenticate to kanto service (currently kanto has only dummy auth, so everyone is able to create clusters, but username is still needed)
 * **token** - string, required: auth token for username, it is similar to password
 
##replicate
path:
`/v0/replicate`

POST values:
 * **cluster_tag** - string,required; couchdb cluster name/tag that will be scaled
 * **databases**  - string,required; list of dbs to replicate in couchdb cluster, delimiter is "," example: mydb1,special,test1 
 * **username** - string, required: username to authenticate to kanto service (currently kanto has only dummy auth, so everyone is able to create clusters, but username is still needed)
 * **token** - string, required: auth token for username, it is similar to password

##list
path:
`/v0/list`

POST values:
 * **username** - string, required: username to authenticate to kanto service (currently kanto has only dummy auth, so everyone is able to create clusters, but username is still needed)
 * **token** - string, required: auth token for username, it is similar to password
 
  
##detail
path:
`/v0/detail`

POST values:
 * **cluster_tag** - string,required; couchdb cluster name/tag
 * **username** - string, required: username to authenticate to kanto service (currently kanto has only dummy auth, so everyone is able to create clusters, but username is still needed)
 * **token** - string, required: auth token for username, it is similar to password
 
##api test examples
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
 
 
 
#Couchdb Cluster configuration
info about how kanto creates couchdb cluster and how it configure replication withtin couchdb

##kubernetes info
kanto works fine with kubernetes 1.2.1+

kanto is using **couchdb** 1.6.1 official docker image for kubernetes pods, couchdb port is default - **5984**

if using **SPAWNER_TYPE=deployment**
Recommended for testing/development. This solution does not use persistent volumes. Replication is configured via pod IP address which can be volatile.
When creating a new couchdb cluster, kanto will create "kind: Deployment" and "kind: Service" in kubernetes. 
Deployment will create corresponding pods (amount is specified with Replicas values).
These pods are not linked with any application logic. 
Service will create endpoint that will be accessible from outside.
This endpoint will load-balance request to all deployment pods.



if using **SPAWNER_TYPE=rc**

When creating a new couchdb cluster, kanto will create  one "kind: Service" and then replication controller for each replica.
Each replication controller spawns 1 pod and each pod has volume mounts and persistent volume claim for this volume.
These pods are not linked with any application logic.
Service will create endpoint that will be accessible from outside.
This endpoint will load-balance request to all deployment pods.
This solution requires pre-created persistent volumes in kubernetes (at least 5Gi storage and "ReadWriteOnce" access mode)
Each pod will have own service, which will be used for replication. (Pod's ip is volatile and can change, service ip/name is always same)

##couchdb pod configuration
Docker image couchdb is started with ENVs **COUCHDB_USER** and **COUCHDB_PASSWORD**  with values username and token.
This configuration will disable couchdb admin party mode and only admin will be able to do privileged operations.
Pod has exposed port 5984 to access couchdb. Persistent volumes (if used) is mounted to "**/usr/local/var/lib/couchdb**".

##replication between pods
The biggest problem with couchdb replication is that it can not be configured for all databases. 
Each database has to be separately configured for replication. This means user has to sent request for each db that should be replicated in cluster.
Unfortunately when scaling (up or down), replication has to be cleared and reconfigured.
This mean we have to save which dbs user want to replicate, so we can load this db list when scaling.
Since this is stateless configuration, this has to be implemented. But it should be fair easy, 
If you have somewhere running database with this information. (just save on /replicate request and load on /scale request)
Now it only stores db information to global variable so every restart will wipe data. Check file **user.go** and functions: **DatabasesToReplicate** and **SaveReplDatabases**


Replication is configured via "_replicator" database and is always **continuous**.
Replication is configured in circle (ie. pod1 replicates to pod2, pod2 replicates to pod3, ... , podN replicates to pod1)
Unfortunately in couchdb 1.6.1 there is a bug that fails replicate database "_users".
Replication will be aborted with message that replication worked died. (in replication message there is actual erlang stacktrace instead of error message).
Same settings in database "_replicate" works.


Couchdb 2.+ offers clustering, but official docker image cannot be used since its wraps everything and starts already clustered couchdb (2+ nodes)
in single docker container listening on localhost and haproxy which balances requests to nodes.

#Limitations
To move it into production I would recommend implement:
 * corresponding authentication - **user.go** has only dummy functions
 * saving information about what databases user want replicate - check section: [replication](#replication-between-pods) for more info
 * sophisticated couchdb username and password configuration - currently, each couchdb will have admin user created with username:token credentials
