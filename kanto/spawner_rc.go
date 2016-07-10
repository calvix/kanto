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
	podTemplate := cluster.CouchdbPodTemplate(true, pvClaim.Name)
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
				return err
			} else {
				DebugLog("spawner_rc: created replication controller: "+rc.Name)
			}
			// create service for pod
			svc, err := cluster.CreatePodService(cluster.Labels)
			if err != nil {
				ErrorLog("spawner_rc: createCLuster: failed to create service for pod "+strconv.Itoa(i))
				ErrorLog(err)
				return err
			} else {
				DebugLog("spawner_rc: created pod service: "+svc.Name)
			}

		}
	}
	delete(cluster.Labels, LABEL_REPLICA)
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
	// pod service deletion

	// servicePod labels
	serviceLabels := make(map[string]string)
	for k,v := range cluster.Labels {
		serviceLabels[k] = v
	}
	serviceLabels[LABEL_POD_SERVICE] = "true"
	// special list options for pod service
	listOptions = api.ListOptions{LabelSelector: labels.SelectorFromSet(labels.Set(serviceLabels))}
	// delete all pod services
	svcList, err := c.Services(cluster.Namespace).List(listOptions)
	if err != nil {
		ErrorLog("spawner_rc: delete podSvc: list svc error")
		return err
	}
	// iterate thorough all RC
	for _, svc := range svcList.Items {
		err = c.Services(cluster.Namespace).Delete(svc.Name)
		if err != nil {
			ErrorLog("spawner_rc: delete podSvc: delete svc error")
			return err
		} else {
			DebugLog("spawner_rc: deleted podSvc : "+svc.Name)
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
			ErrorLog("spawner_rc : ScaleRCDown: list RC error")
			return err
		} else {
			// delete
			c.ReplicationControllers(cluster.Namespace).Delete(rcs.Items[0].Name)
			DebugLog("spawner_rc: ScaleRCDown: deleted replicaion controller: "+rcs.Items[0].Name)
		}
		// delete orphaned PVC
		pvc, err := c.PersistentVolumeClaims(cluster.Namespace).List(listOptions)
		if err != nil {
			ErrorLog("spawner_rc : ScaleRCDown: list pvc error")
			return err
		} else {
			// delete
			c.PersistentVolumeClaims(cluster.Namespace).Delete(pvc.Items[0].Name)
			DebugLog("spawner_rc: ScaleRCDown: deleted pvc "+pvc.Items[0].Name)
		}
		//delete orphaned POD
		pod, err := c.Pods(cluster.Namespace).List(listOptions)
		if err != nil {
			ErrorLog("spawner_rc: ScaleRCDown: list pvc error")
			return err
		} else {
			// delete
			c.Pods(cluster.Namespace).Delete(pod.Items[0].Name, &deleteOptions)
			DebugLog("spawner_rc: ScaleRCDown: deleted pod "+pod.Items[0].Name)
		}

		//delete pod service
		err = cluster.DeletePodService(cluster.Labels)
		if err != nil {
			ErrorLog("spawner_rc: ScaleRCDown: delete pod service error")
			return err
		} else {
			DebugLog("spawner_rc: ScaleRCDown: deleted pod service")
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
		}else {
			DebugLog("spawner_rc: created replica controller: "+rc.Name)
		}
		// create service for pod
		svc, err := cluster.CreatePodService(cluster.Labels)
		if err != nil {
			ErrorLog("spawner_rc: createCLuster: failed to create service for pod "+strconv.Itoa(i))
			ErrorLog(err)
			return err
		} else {
			DebugLog("spawner_rc: created pod service: "+svc.Name)
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

// create headless service for pod, so tat we can use service ip instead of pods volatile IP
// @param selector - selector that  will find corresponding pod, this should have label "replica" set
// @return *api.Service - created service
// @return error
func (cluster *CouchdbCluster) CreatePodService(selector map[string]string) (*api.Service,error) {
	// service special label
	serviceLabels := make(map[string]string)
	for k,v := range selector {
		serviceLabels[k] =v
	}
	// add special label
	serviceLabels[LABEL_POD_SERVICE] = "true"

	// svc port
	svcPorts := api.ServicePort{Port: COUCHDB_PORT}
	// service specs
	serviceSpec := api.ServiceSpec{Selector: selector, Ports: []api.ServicePort{svcPorts},/* ClusterIP: "None"*/}
	// init service struct
	service := api.Service{Spec: serviceSpec}
	service.GenerateName = cluster.Tag + "-pod-"
	service.Labels = serviceLabels
	// get a new kube client
	c, err := KubeClient(KUBE_API)
	// check for errors
	if err != nil {
		ErrorLog("spawner_rc: CreatePodService: Cannot connect to Kubernetes api ")
		ErrorLog(err)
		return nil, err
	} else {
		// create service in namespace
		return c.Services(cluster.Namespace).Create(&service)
	}
}

// delete headless service for pod, so tat we can use service dns name instead of pods volatile IP
// @param selector - selector that  will find corresponding service to delete, should have label "replica" set
// @return *api.Service - created service
// @return error
func (cluster *CouchdbCluster) DeletePodService(selector map[string]string) (error) {
	// list options
	listOptions := api.ListOptions{LabelSelector: labels.SelectorFromSet(labels.Set(selector))}
	// get a new kube client
	c, err := KubeClient(KUBE_API)
	// check for errors
	if err != nil {
		ErrorLog("spawner_rc: Delete pod Service: Cannot connect to Kubernetes api ")
		ErrorLog(err)
		return err
	} else {
		//  get list service
		svcList, err := c.Services(cluster.Namespace).List(listOptions)
		if err != nil {
			ErrorLog("spawner_rc: Delete pod Service: get svc list fail ")
			ErrorLog(err)
			return err
		}
		// there should be only one element
		return c.Services(cluster.Namespace).Delete(svcList.Items[0].Name)
	}
}

// get all pod services
func (cluster *CouchdbCluster)GetAllPodServices() (*[]api.Service, error) {
	// service special label
	serviceLabels := make(map[string]string)
	for k,v := range cluster.Labels {
		serviceLabels[k] =v
	}
	// add special pod-service label
	serviceLabels[LABEL_POD_SERVICE] = "true"

	// list options
	listOptions := api.ListOptions{LabelSelector: labels.SelectorFromSet(labels.Set(serviceLabels))}

	// get a new kube client
	c, err := KubeClient(KUBE_API)
	// check for errors
	if err != nil {
		ErrorLog("spawner_rc: get all pod Service: Cannot connect to Kubernetes api ")
		ErrorLog(err)
		return nil, err
	} else {
		//  get list service
		svcList, err := c.Services(cluster.Namespace).List(listOptions)
		if err != nil {
			ErrorLog("spawner_rc: get all pod Service: get svc list fail ")
			ErrorLog(err)
			return nil, err
		}
		// fine, no errors
		return &svcList.Items, nil
	}
}