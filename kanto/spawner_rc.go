package kanto

import (
	"strconv"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/labels"
	"strings"
	"errors"
	"k8s.io/kubernetes/pkg/api/resource"
)

// init replication controller struct and fill it with specs
//  also creates pvc claim for pod
// @param cluster *CouchdbCluster
// @return *api.ReplicationController - initialized replication controller
// @return error
func (cluster *CouchdbCluster) CouchdbReplicationController() (*api.ReplicationController,error){
	// kube client
	c, err := KubeClient(KUBE_API)
	if err != nil {
		ErrorLog("spawner_rc: createCluster: failed to get kube api")
		ErrorLog(err)
		return  nil, err
	}
	// get new pv claim
	pvClaim := cluster.CouchdbPVClaim()
	// create new pv claim
	pvClaim, err = c.PersistentVolumeClaims(cluster.Namespace).Create(pvClaim)
	if err != nil {
		ErrorLog("spawner_rc: createCluster: failed create pvc")
		ErrorLog(err)
		return  nil, err
	} else {
		// OK
		DebugLog("spawner_rc: created pvc "+pvClaim.Name)
	}
	// get pod template for replication controller
	podTemplate := cluster.CouchdbPodTemplate(false, pvClaim.Name)
	// replication controller spec
	rcSpec := api.ReplicationControllerSpec{Selector: cluster.Labels, Template: podTemplate,
								Replicas: 1}
	// replication controller
	rc := &api.ReplicationController{Spec: rcSpec}
	rc.GenerateName = CLUSTER_PREFIX + cluster.Tag + "-"
	rc.Labels = cluster.Labels
	//no errors return rc
	return rc, nil
}

// create couchdb clusters with replica controllers and persistent volumes
// create rc controller and pvc for each cluster.Replica
// init all necessary struct for rc and pvc and then via kube client creates it
// @param cluster - struct CouchdbCluster with filled data
// @return error - errors that occur during creation
func (cluster *CouchdbCluster) CreateReplicationControllers() (error) {
	c, err := KubeClient(KUBE_API)
	// check for errors
	if err != nil {
		ErrorLog("spawner_rc: create rc with PV: Cannot connect to Kubernetes api ")
		ErrorLog(err)
		return  err
	} else {
		replicas := int(cluster.Replicas)
		// create replication controller for each replica
		for i:=0 ; i < replicas; i++ {
			// add replica label for rc controller
			cluster.Labels[LABEL_REPLICA] = strconv.Itoa(i)
			// init replication controller
			rc, err := cluster.CouchdbReplicationController()
			// create replication controller
			rc, err = c.ReplicationControllers(cluster.Namespace).Create(rc)
			if err != nil {
				ErrorLog("spawner_rc: createCLuster: failed to create replication controller "+strconv.Itoa(i))
				ErrorLog(err)
				// TODO delete replication controllers and pvc
				return err
			} else {
				DebugLog("spawner_rc: created replication controller: "+rc.Name)
			}
		}
	}
	// everything is OK
	return nil
}

// tries get ReplicationControllers from kubernetes with specified cluster tag
// @return *[]api.ReplicationController - found rc array, return nil if rc  was not found
// @return error - any error that occurs during fetching rc
func (cluster *CouchdbCluster) GetReplicationControllers() (*[]api.ReplicationController, error) {
	// get kube extensions api
	c, err := KubeClient(KUBE_API)
	// check for errors
	if err != nil {
		ErrorLog("spawner_rc: getReplicationControllers: Cannot connect to Kubernetes api ")
		ErrorLog(err)

		return nil, err
	} else {
		// list options
		listOptions := api.ListOptions{LabelSelector:  labels.SelectorFromSet(labels.Set(cluster.Labels))}
		// get all deployments for this user
		rcList, err := c.ReplicationControllers(cluster.Namespace).List(listOptions)
		if err != nil {
			ErrorLog("spawner_rc: getReplicationControllers:  getReplicationCOntrollers list error ")
			ErrorLog(err)

			return nil, err
		}
		return &rcList.Items, nil
	}
	// nothing matches, return fail
	return nil,  errors.New("spawner_rc: deployment not found")
}

// Delete all replication controllers for this couchdb cluster
// deletes replication controllers and pvc
// @param cluster *CouchdbCluster - cluster that will be deleted
// @return error
func (cluster *CouchdbCluster) DeleteReplicationControllers() (error) {
	// get kube client
	c, err := KubeClient(KUBE_API)
	if err != nil {
		ErrorLog("spawner_rc : delete rcs: kube client error")
		return err
	}
	// list options, with label selector
	listOptions := api.ListOptions{LabelSelector: labels.SelectorFromSet(labels.Set(cluster.Labels))}

	// delete all RC for this cluster
	// get RC list
	rcList, err := c.ReplicationControllers(cluster.Namespace).List(listOptions)
	if err != nil {
		ErrorLog("spawner_rc: delete rcs: list rc error")
		return err
	}
	// iterate thorough all RC
	for _, rc := range rcList.Items {
		// save-guard
		if strings.HasPrefix(rc.Name, CLUSTER_PREFIX + cluster.Tag) {
			// delete RC
			err = c.ReplicationControllers(cluster.Namespace).Delete(rc.Name)
			if err != nil {
				ErrorLog("spawner_rc: delete rcs: delete rc error")
				return err
			} else {
				DebugLog("spawner_rc deleted replication controller: "+rc.Name)
			}
		}
	}
	// delete all pvc
	pvcList, err := c.PersistentVolumeClaims(cluster.Namespace).List(listOptions)
	if err != nil {
		ErrorLog("spawner_rc: delete rcs: list pvc error")
		return err
	}
	// iterate thorough all RC
	for _, pvc := range pvcList.Items {
		// save-guard
		if strings.HasPrefix(pvc.Name, CLUSTER_PREFIX + cluster.Tag) {
			// delete RC
			err = c.PersistentVolumeClaims(cluster.Namespace).Delete(pvc.Name)
			if err != nil {
				ErrorLog("spawner_rc: delete pvc: delete pvc error")
				return err
			} else {
				DebugLog("spawner_rc: deleted pvc : "+pvc.Name)
			}
		}
	}
	// no errors
	return nil
}

// scale couchdb cluster to new replica number
// @param cluster *CouchdbCluster - coucbdb cluster with new replica number
// @param rcList *[]api.ReplicationController - lsit of current rc - fetched via GetReplicationCOntrollers()
// @return error - error
func (cluster *CouchdbCluster) ScaleRC(rcList *[]api.ReplicationController) (error){
	// old and new replica count
	currentReplicas := len(*rcList)
	newReplicas := int(cluster.Replicas)
	//DebugLog("old replica count:"+strconv.Itoa(currentReplicas)+", new replica count:"+strconv.Itoa(newReplicas))

	var err error
	// scale up or down ?
	if newReplicas > currentReplicas {
		// scape up
		DebugLog("spawner_rc: scaleRC: Scaling Up")
		err = cluster.ScaleRCup(newReplicas, currentReplicas)

	} else if newReplicas < currentReplicas {
		// scale down
		DebugLog("spawner_rc: scaleRC: Scaling Down")
		err = cluster.ScaleRCDown(newReplicas, currentReplicas)
	} else {
		// newReplicas == currentReplicas
		//  nothing to do
		return nil
	}
	// check for errors
	if err != nil {
		ErrorLog("spawner_rc: ScaleRC: scale error")
		return err
	}

	// we need to reconfigure replication
	err = cluster.SetupReplication(DatabasesToReplicate(cluster.Username))
	if err != nil {
		ErrorLog("spawner_rc : ScaleRC: reconfigure replication error")
		return err
	}
	//everything OK
	return nil
}
// scale RC down to new replica number
// @param cluster *CouchdbCluster - coucbdb cluster with new replica number
// @param newReplicas int - new number for replicas
// @param currentReplicas int - old number for replicas
// @return error - error
func (cluster *CouchdbCluster) ScaleRCDown(newReplicas int, currentReplicas int) (error) {
	// get kube extensions client
	c, err := KubeClient(KUBE_API)
	if err != nil {
		ErrorLog("spawner_rc : ScaleRCDown: kube extensions client error")
		return err
	}
	// options for delete
	orphan := true
	deleteOptions := api.DeleteOptions{OrphanDependents: &orphan}
	// SCALE DOWN, delete extra rc
	// how much rc we should delete
	replica_difference := currentReplicas - newReplicas

	for i:= 0 ; i < replica_difference; i++ {
		// odl replica index, this rc will be deleted
		oldReplicaIndex :=  currentReplicas - 1 - i
		cluster.Labels[LABEL_REPLICA]= strconv.Itoa(oldReplicaIndex)
		// list options, with label selector
		listOptions := api.ListOptions{LabelSelector: labels.SelectorFromSet(labels.Set(cluster.Labels))}
		// RC
		// get last RC, use label selector for filter only rc with specified number
		rcs, err := c.ReplicationControllers(cluster.Namespace).List(listOptions)
		if err != nil {
			ErrorLog("spawner_rc : ScaleRC: list RC error")
			return err
		} else {
			// delete
			c.ReplicationControllers(cluster.Namespace).Delete(rcs.Items[0].Name)
			DebugLog("spawner_rc: deleted replicaion controller: "+rcs.Items[0].Name)
		}
		// delete orphaned PVC
		pvc, err := c.PersistentVolumeClaims(cluster.Namespace).List(listOptions)
		if err != nil {
			ErrorLog("spawner_rc : ScaleRC: list pvc error")
			return err
		} else {
			// delete
			c.PersistentVolumeClaims(cluster.Namespace).Delete(pvc.Items[0].Name)
			DebugLog("spawner_rc: deleted pvc "+pvc.Items[0].Name)
		}
		//delete orphaned POD
		pod, err := c.Pods(cluster.Namespace).List(listOptions)
		if err != nil {
			ErrorLog("spawner_rc: ScaleRC: list pvc error")
			return err
		} else {
			// delete
			c.Pods(cluster.Namespace).Delete(pod.Items[0].Name, &deleteOptions)
			DebugLog("spawner_rc: deleted pod "+pod.Items[0].Name)
		}
	 }
	// clear labels
	delete(cluster.Labels, LABEL_REPLICA)
	// no errors
	return nil
}
// scale RC Up to new replica number
// @param cluster *CouchdbCluster - coucbdb cluster with new replica number
// @param newReplicas int - new number for replicas
// @param currentReplicas int - old number for replicas
// @return error - error
func (cluster *CouchdbCluster) ScaleRCup(newReplicas int, currentReplicas int) (error){
	// get kube extensions client
	c, err := KubeClient(KUBE_API)
	if err != nil {
		ErrorLog("spawner_rc : ScaleRCDown: kube extensions client error")
		return err
	}
	// how much new rc should we spawn
	replica_difference := newReplicas - currentReplicas

	for i:= 0 ; i < replica_difference; i++ {
		// new replica index
		newReplicaIndex := currentReplicas + i
		// add replica index label
		cluster.Labels[LABEL_REPLICA]= strconv.Itoa(newReplicaIndex)
		// init rc
		rc , err := cluster.CouchdbReplicationController()
		if err != nil{
			ErrorLog("spawner_rc: scale rc: failed to init replication controller ")
			ErrorLog(err)
			return err
		}
		// create replication controller
		rc, err = c.ReplicationControllers(cluster.Namespace).Create(rc)
		if err != nil {
			ErrorLog("spawner_rc: scale rc: failed to create replication controller "+strconv.Itoa(i))
			ErrorLog(err)
			return err
		}
	}
	// clear labels
	delete(cluster.Labels, LABEL_REPLICA)

	return nil
}

// create pvc claim for couchdb pod, used for rc spawner
// pvc name is automatically generated by kubernetes
// @param cluster *CouchdbCluster - cluster that will be using this pvc
// @return *api.PersistentVolumeClaim - filled pvc claim struct ready to be created
func (cluster *CouchdbCluster) CouchdbPVClaim() (*api.PersistentVolumeClaim){
	// resource list
	rsList := make(api.ResourceList)
	// SIZE
	rsList[api.ResourceStorage] = *(resource.NewQuantity(COUCHDB_VOLUME_SIZE, resource.BinarySI))

	// pvc SPEC, witch readWriteOnce access mode
	pvcSpec := api.PersistentVolumeClaimSpec{AccessModes: []api.PersistentVolumeAccessMode{api.ReadWriteOnce}}
	pvcSpec.Resources.Requests = api.ResourceList(rsList)
	// PVC
	pvc := api.PersistentVolumeClaim{Spec:pvcSpec}
	pvc.GenerateName = CLUSTER_PREFIX + cluster.Tag + "-"
	pvc.Labels = cluster.Labels

	return &pvc
}