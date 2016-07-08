package kanto

import (
	"k8s.io/kubernetes/pkg/apis/apps"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/resource"
)

/*
	PET SET, included in kube 1.3+
	http://kubernetes-v1-3.github.io/docs/user-guide/petset/
	// TESTING - ALPHA
*/

// PetSet - available only in kubernetes v1.3+
// big advantage: it allow create persistentVolume template
// this template can create persistent volumeClaim for each POD automatically
//
// create petSet for couchdb cluster
// init all necessary struct for petSet and then via kube client creates it
// @ param cluster - struct CouchdbCluster - required:
// @return extensions.Deployment - created kube deployment
// @return error - errors that occur during creation
//
func (cluster * CouchdbCluster) CreatePetSet() (*apps.PetSet, error) {
	/*
	PET SET, included in kube 1.3+
	http://kubernetes-v1-3.github.io/docs/user-guide/petset/
	// TESTING - ALPHA
	*/
	// pod template with volumes
	podTemplate := *cluster.CouchdbPodTemplate(true, CLUSTER_PREFIX+cluster.Tag)
	// pet set spec label selector
	lSelector := unversioned.LabelSelector{MatchLabels: cluster.Labels}

	// pvc claim
	pvc := api.PersistentVolumeClaim{}
	pvc.Name = CLUSTER_PREFIX + cluster.Tag
	pvc.Annotations = make(map[string]string)
	pvc.Annotations["volume.alpha.kubernetes.io/storage-class"] = "anything"

	// resource list for pvc claim template
	rsList := make(api.ResourceList)
	// SIZE
	rsList[api.ResourceStorage] = *(resource.NewQuantity(5*1024*1024*1024, resource.BinarySI))
	// pvc SPEC
	pvcSpec := api.PersistentVolumeClaimSpec{}
	pvcSpec.Resources.Requests = api.ResourceList(rsList)

	pvc.Spec = pvcSpec
	// pet set specs
	petSetSPec := apps.PetSetSpec{Replicas: int(cluster.Replicas), Template: podTemplate,
				Selector: &lSelector, VolumeClaimTemplates: []api.PersistentVolumeClaim{pvc}}

	// pet set
	petSet := apps.PetSet{Spec:petSetSPec}
	petSet.Name = CLUSTER_PREFIX + cluster.Tag
	petSet.Labels = cluster.Labels

	// get a new kube extensions client
	c, err := KubeClientApps(KUBE_API)
	// check for errors
	if err != nil {
		ErrorLog("kube_control: createPetSet: Cannot connect to Kubernetes api ")
		ErrorLog(err)
		return nil, err
	} else {
		// create deployment
		return c.PetSets(cluster.Namespace).Create(&petSet)
	}
}
