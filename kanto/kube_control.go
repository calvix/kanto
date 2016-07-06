// Kanto
// web service to manage and scale couchdb running on kubernetes
// author: Vaclav Rozsypalek
// Created on 21.06.2016

// file for working with kubernetes api
package kanto

import (
	"errors"
	"strings"
	"time"
	// kubernetes imports
	"k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/api"
	client "github.com/kubernetes/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/labels"
)

// default kube api,  can be overwritten by os ENV "KUBERNETES_API_URL"
var KUBE_API string = "http://127.0.0.1:8080"
const (


	COUCHDB_PORT = 5984
	COUCHDB_PORT_STRING = "5984"

	CLUSTER_PREFIX = "cdb-cluster-"
	MAX_REPLICAS = 10
	MAX_CLUSTER_TAG = 12
	MIN_CLUSTER_TAG = 4

	LABEL_USER = "user"
	LABEL_CLUSTER_TAG = "cluster_tag"

	DOCKER_IMAGE = "couchdb"
)

// create kubernetes api client
// @param host string - url for kubernetes API
// @return client - kubernetes api client
// @return error
func KubeClient(host string) (*client.Client, error) {
	// create configuration for kube client
	config :=  &restclient.Config{Host:host}
	// return client and error
	return client.New(config)
}
// create kubernetes api client
// @param host string - url for kubernetes API
// @return client - kubernetes api client
// @return error
func KubeClientExtensions(host string) (*client.ExtensionsClient, error) {
	// create configuration for kube client
	config :=  &restclient.Config{Host:host}
	// return client and error
	return client.NewExtensions(config)
}

// create deployment for couchdb cluster
// init all necessary struct for deployment and then via kube client creates it
// @ param cluster - struct CouchdbCluster - required:
// @return extensions.Deployment - created kube deployment
// @return error - errors that occur during creation
//
func CreateDeployment(cluster * CouchdbCluster)(*extensions.Deployment, error) {

	// container ports init
	contPort := api.ContainerPort{ContainerPort: COUCHDB_PORT}

	// container env init
	contEnv_dbName := api.EnvVar{Name: "COUCHDB_USER", Value: cluster.Username}
	contEnv_dbPass := api.EnvVar{Name: "COUCHDB_PASSWORD", Value: cluster.Password}

	// container specs
	container := api.Container{Name: DOCKER_IMAGE + "-"+ cluster.Tag, Image: DOCKER_IMAGE,
				Ports: []api.ContainerPort{contPort}, Env: []api.EnvVar{contEnv_dbName, contEnv_dbPass}}

	// pod specifications
	podSpec := api.PodSpec{Containers:[]api.Container{container}}

	// pod template spec
	podTemplate := api.PodTemplateSpec{Spec: podSpec}
	podTemplate.Labels = cluster.Labels

	// deployment spec label selector
	lSelector := unversioned.LabelSelector{MatchLabels: cluster.Labels}

	// deployment specification init
	deploymentSpec := extensions.DeploymentSpec{Template:podTemplate, Replicas:cluster.Replicas,Selector:&lSelector }

	// TOP LEVEL entity
	// deployment
	deployment := extensions.Deployment{Spec: deploymentSpec}
	deployment.Name = CLUSTER_PREFIX + cluster.Tag
	deployment.Labels = cluster.Labels

	// get a new kube extensions client
	c, err := KubeClientExtensions(KUBE_API)
	// check for errors
	if err != nil {
		ErrorLog("Cannot connect to Kubernetes api ")
		ErrorLog(err)

		return nil, err
	} else {
		// debug output
		//data, _ := deployment.Marshal()
		//InfoLog(data)

		// create deployment
		return c.Deployments(cluster.Namespace).Create(&deployment)
	}
}
// create service for couchdb cluster
// init all necessary struct for deployment and then via kube client creates it
// service expose cluster to outside
// @param cluster - struct CouchdbCluster - required:tag, labels, namespace, username,
// @return api.Service - created kube service
// @return error - errors that occur during creation
//
func CreateService(cluster * CouchdbCluster) (*api.Service, error) {

	// service ports
	svcPorts := api.ServicePort{Port: COUCHDB_PORT}

	// service specs
	serviceSpec := api.ServiceSpec{Selector: cluster.Labels, Ports: []api.ServicePort{svcPorts}}

	// init service struct
	service := api.Service{Spec: serviceSpec}
	service.Name = CLUSTER_PREFIX + cluster.Tag
	service.Labels = cluster.Labels
	// get a new kube client
	c, err := KubeClient(KUBE_API)
	// check for errors
	if err != nil {
		ErrorLog("kube_control: CreateService: Cannot connect to Kubernetes api ")
		ErrorLog(err)

		return nil, err
	} else {
		// debug output
		//data, _ := service.Marshal()
		//InfoLog(data)

		// create service in namespace
		return c.Services(cluster.Namespace).Create(&service)
	}

}

// tries gets deployment from kubernetes with specified cluster tag
// @param cluster * CouchdbCluster
// @return *extensions.Deployment - found deployment, return nil if deployment was not found
// @return error - any error that occurs during fetching deployment
//
func GetDeployment(cluster * CouchdbCluster) (*extensions.Deployment, error) {
	// get kube extensions api
	c, err := KubeClientExtensions(KUBE_API)
	// check for errors
	if err != nil {
		ErrorLog("kube control: getDeployment: Cannot connect to Kubernetes api ")
		ErrorLog(err)

		return nil, err
	} else {
		// list options
		listOptions := api.ListOptions{LabelSelector:  labels.SelectorFromSet(labels.Set(cluster.Labels))}
		// get all deployments for this user
		deploymentList, err := c.Deployments(cluster.Namespace).List(listOptions)
		if err != nil {
			ErrorLog("kube control: getDeployment: Get deployment list error ")
			ErrorLog(err)

			return nil, err
		}
		// check deployment list, iterate thought all  deployments for this user
		for _, deployment := range deploymentList.Items {
			if deployment.Name == CLUSTER_PREFIX + cluster.Tag {
				// we got match !
				return &deployment, nil
			}
		}
	}
	// nothing matches, return fail
	return nil,  errors.New("deployment not found")
}

// tries gets service from kubernetes with specified cluster tag
// @param cluster * CouchdbCluster
// @return *api.Service - found deployment, return nil if deployment was not found
// @return error - any error that occurs during fetching deployment
//
func GetService(cluster * CouchdbCluster) (*api.Service, error) {
	// get kube api
	c, err := KubeClient(KUBE_API)
	// check for errors
	if err != nil {
		ErrorLog("kube control: GetService: Cannot connect to Kubernetes api ")
		ErrorLog(err)

		return nil, err
	} else {
		// list options
		listOptions := api.ListOptions{LabelSelector:  labels.SelectorFromSet(labels.Set(cluster.Labels))}

		// get all deployments for this user
		serviceList, err := c.Services(cluster.Namespace).List(listOptions)
		if err != nil {
			ErrorLog("kube control: GetService: Get service list  error ")
			ErrorLog(err)

			return nil, err
		}
		// check service list, iterate thought all  services for this user
		for _, service := range serviceList.Items {
			if service.Name == CLUSTER_PREFIX + cluster.Tag {
				// we got match !
				return &service, nil
			}
		}
	}
	// nothing matches, return fail
	return nil,  errors.New("service not found")
}

// tries gets pods from kubernetes with specified cluster tag
// @param cluster * CouchdbCluster
// @return []*api.Pod - all pods that belong to this couchdb cluster
// @return error - any error that occurs during fetching deployment
//
func GetPods(cluster *CouchdbCluster) (*[]api.Pod, error) {
	// get kube api
	c, err := KubeClient(KUBE_API)
	// check for errors
	if err != nil {
		ErrorLog("kube control: GetPods: Cannot connect to Kubernetes api ")
		ErrorLog(err)

		return nil, err
	} else {
		// list options
		listOptions := api.ListOptions{LabelSelector:  labels.SelectorFromSet(labels.Set(cluster.Labels))}

		// get all deployments for this user
		podList, err := c.Pods(cluster.Namespace).List(listOptions)
		if err != nil {
			ErrorLog("kube control: getPods: Get pods list  error ")
			ErrorLog(err)
			return nil, err
		}
		// return pod list
		return &(podList.Items), nil

	}
	// nothing matches, return fail
	return nil,  errors.New("no pods found")
}


// create couchdb cluster and expose it
// will create deployment and service components
// if cluster has more than 1 replica it will setup replication between all pods
// @param cluster - CouchdbCluster struct - required: tag, username, replicas, labels
//
func CreateCouchdbCluster(cluster *CouchdbCluster) (error){
	// create deployment for couchdb
	_, err := CreateDeployment(cluster)

	// check for errors
	if err != nil {
		ErrorLog("kube_control: CreateCouchdbCluster: deployment creating fail")
		ErrorLog(err)
		return err
	}
	// expose couchdb via service
	svc, err := CreateService(cluster)
	// save endpoint to struct
	cluster.Endpoint = ClusterEndpoint(svc.Spec.ClusterIP)
	// check for errors
	if err != nil {
		// TODO, if service creation fails, delete deployment

		ErrorLog("kube_control: CreateCouchdbCluster: service expose creating fail")
		ErrorLog(err)
		return err
	}

	// if required more than 1 replica, configure replication
	if cluster.Replicas > 1 {
		// setup replication for basic databases
		SetupReplication(cluster, []string{"test", "_users"})
	} else {
		DebugLog("not setting replication, only 1 replica")
		DebugLog(cluster.Replicas)
	}
	// no error
	return  nil
}

// delete whole couch db cluster from kubernetes
// function will delete deployment and service and then all orphaned components (cascade deleting does not work)
// @param cluster - CouchdbCluster struct with info about cluster we want delete (required: namespace, tag, username, labels)
// @return error -  error if something goes wrong
func DeleteCouchdbCluster(cluster *CouchdbCluster) (error) {
	// list options, with label selector
	listOptions := api.ListOptions{LabelSelector: labels.SelectorFromSet(labels.Set(cluster.Labels))}

	// options for delete
	orphan := true
	deleteOptions := api.DeleteOptions{OrphanDependents: &orphan}

	// get kube extensions client
	c, err := KubeClientExtensions(KUBE_API)
	if err != nil {
		ErrorLog("kube control : delete coucdb cluster: kube extensions client error")
		return err
	}

	// delete deployment
	err = c.Deployments(cluster.Namespace).Delete(CLUSTER_PREFIX+cluster.Tag, &deleteOptions)
	if err != nil {
		ErrorLog("kube control : delete coucdb cluster: delete deployment error")
		return err
	}

	// get kube client
	c2, err := KubeClient(KUBE_API)

	if err != nil {
		ErrorLog("kube control : delete coucdb cluster: kube client error")
		return err
	}
	// delete service
	err = c2.Services(cluster.Namespace).Delete(CLUSTER_PREFIX+cluster.Tag)
	if err != nil {
		ErrorLog("kube control : delete coucdb cluster: delete service error")
		return err
	}

	// DELETE orphaned kube components
	// cascade deleting is not working, we have to manually delete replica sets and pods

	// delete orphaned replica sets
	// get replica sets list
	replicaSetLists, err := c2.ReplicaSets(cluster.Namespace).List(listOptions)
	if err != nil {
		ErrorLog("kube control : delete coucdb cluster: list replica sets error")
		return err
	}
	// iterate thorough all replica sets and find matching
	for _, replicaSet := range replicaSetLists.Items {
		// check matching name
		if strings.HasPrefix(replicaSet.Name, CLUSTER_PREFIX + cluster.Tag) {
			// we got our replica set, so delete it
			err = c2.ReplicaSets(cluster.Namespace).Delete(replicaSet.Name, &deleteOptions)
			if err != nil {
				ErrorLog("kube control : delete coucdb cluster: delete replica set error")
				return err
			}
			// there is only 1 replica set for each cluster db, so break loop
			break
		}
	}
	// wait for RS to be deleted, so it wont spawn another pods,
	// there should be something better than hardcoded wait,
	// some checks for kubernetes rs
	time.Sleep(time.Millisecond*600)

	// delete orphaned pods
	// get pod list
	podsList, err := c2.Pods(cluster.Namespace).List(listOptions)
	if err != nil {
		ErrorLog("kube control : delete coucdb cluster: list pods error")
		return err
	}
	// iterate thorough all pods and find matching
	for _, pod := range podsList.Items {
		// check matching name
		if strings.HasPrefix(pod.Name, CLUSTER_PREFIX + cluster.Tag) {
			// got matching pod, delete it
			err = c2.Pods(cluster.Namespace).Delete(pod.Name, &deleteOptions)
			if err != nil {
				ErrorLog("kube control : delete coucdb cluster: delete pod error")
				return err
			}
			// there can  be more pod  for each cluster db, so continue loop
		}
	}
	// no error
	return nil
}

// list all couchdb clusters for user
// @param username - string
// @return *[]CouchdbCLuster - array of all couchdb clusters

func ListCouchdbClusters(username string, namespace string) (*[]CouchdbCluster) {
	// result array
	clusters :=  []CouchdbCluster{}

	// get kube extensions api
	c, err := KubeClientExtensions(KUBE_API)
	if err != nil {
		ErrorLog("kube control; listCouchdbclusters: get kube client error")
		ErrorLog(err)
		return nil
	}
	// list options
	userLabels := make(map[string]string)
	userLabels[LABEL_USER] = username
	listOptions := api.ListOptions{LabelSelector: labels.SelectorFromSet(labels.Set(userLabels))}

	// get all deployments
	deploymentList, err := c.Deployments(namespace).List(listOptions)
	if err != nil {
		ErrorLog("kube control; listCouchdbclusters: get deployments error")
		ErrorLog(err)
		return nil
	}
	// iterate through all deployments
	for _, deployment := range deploymentList.Items {
		// get tag from deployment name
		tag := strings.TrimLeft(deployment.Name, CLUSTER_PREFIX)
		// inti labels for cluster
		labels := make(map[string]string)
		labels[LABEL_USER] = username
		labels[LABEL_CLUSTER_TAG] = tag

		// init cluster struct
		cluster := CouchdbCluster{Tag: tag, Username: username, Namespace: namespace,
					Replicas: deployment.Spec.Replicas, Labels: labels}

		// get service info, for endpoint IP
		service , err := GetService(&cluster)
		if err != nil {
			// cannot get info about service,
		} else {
			// save endpoint info
			cluster.Endpoint = ClusterEndpoint(service.Spec.ClusterIP)
		}
		// ad cluster to array
		clusters = append(clusters, cluster)
	}
	// done, return cluster array
	return &clusters
}

// scale couchdb cluster to new replica number
// @param cluster *CouchdbCluster - coucbdb cluster with new replica number
// @param oldDeployment  *extensions.Deployment - deployment with old replica number,  fetched via GetDeployment()
func ScaleCouchdbCluster(cluster *CouchdbCluster, oldDeployment *extensions.Deployment) (error){
	// get kube extensions client
	c, err := KubeClientExtensions(KUBE_API)
	if err != nil {
		ErrorLog("kube control : ScaleCouchdbCluster: kube extensions client error")
		return err
	}
	// update replica number
	oldDeployment.Spec.Replicas = cluster.Replicas

	// update deployment in kubernetes
	_, err = c.Deployments(cluster.Namespace).Update(oldDeployment)
	if err != nil {
		ErrorLog("kube control : ScaleCouchdbCluster: deployment update error")
		return err
	}

	// we need to reconfigure replication
	err = SetupReplication(cluster, []string{"test", "_users"})
	if err != nil {
		ErrorLog("kube control : ScaleCouchdbCluster: reconfigure replication error")
		return err
	}

	//everything OK
	return nil


}

// return couchdb cluster endpoint URL
// @param clusterIp - cluster ip from kubernetes service
func ClusterEndpoint(clusterIp string) (string) {
	return "http://"+clusterIp+":"+COUCHDB_PORT_STRING
}