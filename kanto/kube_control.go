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
	"k8s.io/kubernetes/pkg/labels"
)

// default kube api,  can be overwritten by os ENV "KUBERNETES_API_URL"
var KUBE_API string = "http://127.0.0.1:8080"
const (
	COUCHDB_PORT = 5984
	COUCHDB_PORT_STRING = "5984"

	CLUSTER_PREFIX = "cdb-clust-"
	MAX_REPLICAS = 10
	MAX_CLUSTER_TAG = 12
	MIN_CLUSTER_TAG = 4

	LABEL_USER = "user"
	LABEL_CLUSTER_TAG = "cluster_tag"
	LABEL_REPLICA = "replica"

	DOCKER_IMAGE = "calvix/couchdb"
	COUCHDB_VOLUME_MOUNTPATH = "/usr/local/var/lib/couchdb"
	COUCHDB_VOLUME_SIZE = 5*1024*1024*1024 // 5GB


	COMPONENT_RC = "rc"
	COMPONENT_DEPLOYMENT = "deployment"
	COMPONENT_PETSET = "petset"
)

var SPAWNER_TYPE string = COMPONENT_DEPLOYMENT

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
// create kubernetes api client
// @param host string - url for kubernetes API
// @return client - kubernetes api client
// @return error
func KubeClientApps(host string) (*client.AppsClient, error) {
	// create configuration for kube client
	config :=  &restclient.Config{Host:host}
	// return client and error
	return client.NewApps(config)
}


// create couchdb cluster and expose it
// will create deployment and service components
// if cluster has more than 1 replica it will setup replication between all pods
// @param cluster - CouchdbCluster struct - required: tag, username, replicas, labels
//
func (cluster *CouchdbCluster) CreateCouchdbCluster() (error){
	// create pod spawner for cluster
	var err error
	if SPAWNER_TYPE == COMPONENT_DEPLOYMENT {
		// deployment does nto work with persisten volumes
		_, err = cluster.CreateDeployment()
	} else if SPAWNER_TYPE == COMPONENT_RC {
		// rc works with persistent volumes
		err = cluster.CreateReplicationControllers()
		// clear replica labels
		delete(cluster.Labels, LABEL_REPLICA)
	} else if SPAWNER_TYPE == COMPONENT_PETSET {
		// create pet sets
		_, err = cluster.CreatePetSet()
	}

	// check for errors
	if err != nil {
		ErrorLog("kube_control: CreateCouchdbCluster: deployment creating fail")
		ErrorLog(err)
		return err
	}

	// expose couchdb via service
	svc, err := cluster.CreateService()
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
		cluster.SetupReplication(DatabasesToReplicate(cluster.Username))
	} else {
		DebugLog("kube_control: not setting replication, only 1 replica")
	}
	// no error
	return  nil
}

// delete whole couch db cluster from kubernetes
// function will delete deployment and service and then all orphaned components (cascade deleting does not work)
// @param cluster - CouchdbCluster struct with info about cluster we want delete (required: namespace, tag, username, labels)
// @return error -  error if something goes wrong
func (cluster *CouchdbCluster) DeleteCouchdbCluster() (error) {
	// Delete service
	err := cluster.DeleteService()
	if err != nil{
		ErrorLog("kube_control: deleteCouchdb cluster: delete service")
	}
	// delete spawner
	if SPAWNER_TYPE == COMPONENT_DEPLOYMENT {
		err = cluster.DeleteDeployment()
	} else if SPAWNER_TYPE == COMPONENT_RC {
		err = cluster.DeleteReplicationControllers()
	} else if SPAWNER_TYPE == COMPONENT_PETSET {
		// TODO petset deletion
	}
	// check for delete errors
	if err != nil{
		ErrorLog("kube_control: deleteCouchdb cluster: delete spawner")
	}
	// wait for spawner to be deleted, so it wont spawn another pods,
	// there should be something better than hardcoded wait,
	// some checks for kubernetes rs
	time.Sleep(time.Millisecond*600)

	// delete all remaining pods
	err = cluster.DeletePods()
	if err != nil {
		ErrorLog("kube_control: delete deployment: delete pods")
		return err
	}
	// no error
	return nil
}

// list all couchdb clusters for user
// @param username - string
// @return *[]CouchdbCLuster - array of all couchdb clusters
func ListCouchdbClusters(username string, namespace string) (*[]CouchdbCluster, error) {
	// result array
	clusters :=  []CouchdbCluster{}

	// get kube extensions api
	c, err := KubeClient(KUBE_API)
	if err != nil {
		ErrorLog("kube control; listCouchdbclusters: get kube client error")
		ErrorLog(err)
		return nil, err
	}
	// list options
	userLabels := make(map[string]string)
	userLabels[LABEL_USER] = username
	listOptions := api.ListOptions{LabelSelector: labels.SelectorFromSet(labels.Set(userLabels))}

	// get all deployments
	serviceList, err := c.Services(namespace).List(listOptions)
	if err != nil {
		ErrorLog("kube control; listCouchdbclusters: get service list error")
		ErrorLog(err)
		return nil, err
	}
	// iterate through all services
	for _, service := range serviceList.Items {
		// get tag from deployment name
		tag := strings.TrimLeft(service.Name, CLUSTER_PREFIX)
		// inti labels for cluster
		labels := make(map[string]string)
		labels[LABEL_USER] = username
		labels[LABEL_CLUSTER_TAG] = tag

		// init cluster struct
		cluster := &CouchdbCluster{Tag: tag, Username: username, Namespace: namespace,
					Endpoint: service.Spec.ClusterIP, Labels: labels}
		// get replica count
		if SPAWNER_TYPE == COMPONENT_DEPLOYMENT {
			// get deployment
			deployment, err := cluster.GetDeployment()
			if err != nil {
				ErrorLog("kube control; listCouchdbclusters: get deployment error")
				ErrorLog(err)
			} else {
				// replica number
				cluster.Replicas = deployment.Spec.Replicas
			}
		} else if SPAWNER_TYPE == COMPONENT_RC {
			// get all replica controllers
			rcList, err := cluster.GetReplicationControllers()
			if err != nil {
				ErrorLog("kube control; listCouchdbclusters: get repl controllers error")
				ErrorLog(err)
			} else {
				// each replication controller means one replica for couchdb cluster
				cluster.Replicas = int32(len(*rcList))
			}
		} else if SPAWNER_TYPE == COMPONENT_PETSET {
			// TODO
		}
		// add cluster to array
		clusters = append(clusters, *cluster)
	}
	// done, return cluster array
	return &clusters, nil
}

// list all couchdb clusters for user
// @param username - string
// @return *[]CouchdbCLuster - array of all couchdb clusters
func (cluster *CouchdbCluster) ScaleCouchdbCluster() (error) {
	var err error

	if SPAWNER_TYPE == COMPONENT_DEPLOYMENT {
		deployment, _ := cluster.GetDeployment()
		// its ok, scale cluster
		err = cluster.ScaleDeployment(deployment)
	} else if SPAWNER_TYPE == COMPONENT_RC {
		//get rrc list
		rcList, _ := cluster.GetReplicationControllers()
		// scale replication controllers
		err = cluster.ScaleRC(rcList)
	}
	return err
}



// create service for couchdb cluster
// init all necessary struct for deployment and then via kube client creates it
// service expose cluster to outside
// @param cluster - struct CouchdbCluster - required:tag, labels, namespace, username,
// @return api.Service - created kube service
// @return error - errors that occur during creation
func (cluster *CouchdbCluster) CreateService() (*api.Service, error) {
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
		// create service in namespace
		return c.Services(cluster.Namespace).Create(&service)
	}
}


// tries gets service from kubernetes with specified cluster tag
// @param cluster * CouchdbCluster
// @return *api.Service - found deployment, return nil if deployment was not found
// @return error - any error that occurs during fetching deployment
func (cluster *CouchdbCluster) GetService() (*api.Service, error) {
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
// delete service for couchdb cluster
// @param cluster *CouchdbCluster - cluster that will be deleted
// @return - error
func (cluster *CouchdbCluster) DeleteService() (error) {
	// get kube client
	c, err := KubeClient(KUBE_API)

	if err != nil {
		ErrorLog("kube control : delete service: kube client error")
		return err
	}
	// delete service
	err = c.Services(cluster.Namespace).Delete(CLUSTER_PREFIX+cluster.Tag)
	if err != nil {
		ErrorLog("kube control : delete service: delete service error")
		return err
	}
	return nil
}

// tries gets pods from kubernetes with specified cluster tag
// @param cluster * CouchdbCluster
// @return []*api.Pod - all pods that belong to this couchdb cluster
// @return error - any error that occurs during fetching deployment
func (cluster *CouchdbCluster) GetPods() (*[]api.Pod, error) {
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

// delete all pods that belongs to couchdb cluster
// uses label selector to find all pods
// @param cluster *CouchdbCluster
// @return error
func (cluster *CouchdbCluster) DeletePods() (error) {
	// get kube client
	c, err := KubeClient(KUBE_API)
	if err != nil {
		ErrorLog("kube control : delete pods: kube client error")
		return err
	}
	// options for delete
	orphan := true
	deleteOptions := api.DeleteOptions{OrphanDependents: &orphan}
	// list options, with label selector
	listOptions := api.ListOptions{LabelSelector: labels.SelectorFromSet(labels.Set(cluster.Labels))}

	// delete orphaned pods
	// get pod list
	podsList, err := c.Pods(cluster.Namespace).List(listOptions)
	if err != nil {
		ErrorLog("kube control : delete pods: list pods error")
		return err
	}
	// iterate thorough all pods and find matching
	for _, pod := range podsList.Items {
		// check matching name
		if strings.HasPrefix(pod.Name, CLUSTER_PREFIX + cluster.Tag) {
			// got matching pod, delete it
			err = c.Pods(cluster.Namespace).Delete(pod.Name, &deleteOptions)
			if err != nil {
				ErrorLog("kube control: delete pods: delete pod error")
				return err
			}
			// there can  be more pod  for each cluster db, so continue loop
		}
	}

	return nil
}

// init podTemplate for kubernetes
// @param cluster *CouchdbCluster - cluster for which podTemplate will be
// @param volumes bool - if true podTemplate will have also configured persistent volumes
// @param pvClaimName string - required only when volumes is true, persistent volume claim that will be bound to this pod
// @return *api.PodTemplateSpec - initialized podTemplateSpec
func (cluster *CouchdbCluster) CouchdbPodTemplate(volumes bool, pvcClaimName string) (*api.PodTemplateSpec) {
	// container ports init
	contPort := api.ContainerPort{ContainerPort: COUCHDB_PORT}

	// container env init
	contEnv_dbName := api.EnvVar{Name: "COUCHDB_USER", Value: cluster.Username}
	contEnv_dbPass := api.EnvVar{Name: "COUCHDB_PASSWORD", Value: cluster.Password}

	// container specs
	container := api.Container{Name: CLUSTER_PREFIX + "-"+ cluster.Tag, Image: DOCKER_IMAGE,
						Ports: []api.ContainerPort{contPort}, Env: []api.EnvVar{contEnv_dbName, contEnv_dbPass}}

	//VOLUMES in container
	if volumes {
		// volume mount
		volMount := api.VolumeMount{Name:CLUSTER_PREFIX + cluster.Tag, MountPath: COUCHDB_VOLUME_MOUNTPATH}
		// !! comment if you do not want to user PV a PVC
		container.VolumeMounts = []api.VolumeMount{volMount}
	}
	// pod specifications
	podSpec := api.PodSpec{Containers:[]api.Container{container}}

	// assign PVC for container
	if volumes {
		// persistent volumes, claims
		pvClaim := api.PersistentVolumeClaimVolumeSource{ClaimName: pvcClaimName}
		volume := api.Volume{Name:CLUSTER_PREFIX + cluster.Tag}
		volume.PersistentVolumeClaim = &pvClaim
		// !! comment if you do not want to user PV a PVC
		podSpec.Volumes = []api.Volume{volume}
	}

	// pod template spec
	podTemplateSpec := api.PodTemplateSpec{Spec: podSpec}
	podTemplateSpec.Labels = cluster.Labels

	return &podTemplateSpec
}