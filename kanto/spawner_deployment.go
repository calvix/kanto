package kanto

import (
	"k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/labels"
	"strings"
	"errors"
	"k8s.io/kubernetes/pkg/api/unversioned"
)


// create deployment for couchdb cluster
// init all necessary struct for deployment and then via kube client creates it
// @ param cluster - struct CouchdbCluster - required:
// @return extensions.Deployment - created kube deployment
// @return error - errors that occur during creation
//
func CreateDeployment(cluster * CouchdbCluster)(*extensions.Deployment, error) {
	// get  pod template without volumes
	podTemplate := *CouchdbPodTemplate(cluster, false, "")

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
		ErrorLog("kube_control: createDeployment: Cannot connect to Kubernetes api ")
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

// tries gets deployment from kubernetes with specified cluster tag
// @param cluster * CouchdbCluster
// @return *extensions.Deployment - found deployment, return nil if deployment was not found
// @return error - any error that occurs during fetching deployment
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

// Delete deployment for specified couchdb cluster
// deletes deployment and replica sets
//@param cluster *CouchdbCluster -
func DeleteDeployment(cluster *CouchdbCluster) (error) {
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
	// DELETE orphaned kube components
	// cascade deleting is not working, we have to manually delete replica sets and pods
	// get kube client
	c2, err := KubeClient(KUBE_API)
	if err != nil {
		ErrorLog("kube control : delete coucdb cluster: kube client error")
		return err
	}
	// delete orphaned replica sets
	// get replica sets list
	// list options, with label selector
	listOptions := api.ListOptions{LabelSelector: labels.SelectorFromSet(labels.Set(cluster.Labels))}
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
	// no errors
	return nil
}

// scale couchdb cluster to new replica number
// @param cluster *CouchdbCluster - coucbdb cluster with new replica number
// @param oldDeployment  *extensions.Deployment - deployment with old replica number,  fetched via GetDeployment()
func ScaleDeployment(cluster *CouchdbCluster, oldDeployment *extensions.Deployment) (error){
	// get kube extensions client
	c, err := KubeClientExtensions(KUBE_API)
	if err != nil {
		ErrorLog("kube control : ScaleDeployment: kube extensions client error")
		return err
	}
	// update replica number
	oldDeployment.Spec.Replicas = cluster.Replicas

	// update deployment in kubernetes
	_, err = c.Deployments(cluster.Namespace).Update(oldDeployment)
	if err != nil {
		ErrorLog("kube control : ScaleDeployment: deployment update error")
		return err
	}

	// we need to reconfigure replication
	err = SetupReplication(cluster, DatabasesToReplicate())
	if err != nil {
		ErrorLog("kube control : ScaleDeployment: reconfigure replication error")
		return err
	}
	//everything OK
	return nil
}