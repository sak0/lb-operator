package controller

import (
	"time"
	"os"
	"reflect"
	
	"github.com/golang/glog"
	
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"	
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	
	crdclient "github.com/sak0/lb-operator/pkg/client"
	crdv1 "github.com/sak0/lb-operator/pkg/apis/loadbalance/v1"
	driver "github.com/sak0/lb-operator/pkg/drivers"
)

type CLBController struct {
	crdClient		*rest.RESTClient
	crdScheme		*runtime.Scheme
	client			kubernetes.Interface
	
	clbController	cache.Controller
	epController	cache.Controller
	
	clbstore     	cache.Store
	epstore			cache.Store
	
	driver			driver.LbProvider			
}

func NewCLBController(client kubernetes.Interface, crdClient *rest.RESTClient, 
					crdScheme *runtime.Scheme)(*CLBController, error) {
	clbctr := &CLBController{
		crdClient 	: crdClient,
		crdScheme 	: crdScheme,
		client		: client,
	}
	driver, _ := driver.New("citrix")
	clbctr.driver = driver
	
	//Construction CLB Informer
	clbListWatch := cache.NewListWatchFromClient(clbctr.crdClient, 
		crdv1.CLBPlural, meta_v1.NamespaceAll, fields.Everything())
	
	clbstore, clbcontroller := cache.NewInformer(
		clbListWatch,
		&crdv1.ClassicLoadBalance{},
		time.Minute*10,
		cache.ResourceEventHandlerFuncs{
			AddFunc: clbctr.onClbAdd,
			DeleteFunc: clbctr.onClbDel,
			UpdateFunc: clbctr.onClbUpdate,
		},
	)
	clbctr.clbController = clbcontroller
	clbctr.clbstore = clbstore
	
	//Construction Endpoint Informer
	epListWatch := cache.NewListWatchFromClient(client.Core().RESTClient(), 
		"endpoints", meta_v1.NamespaceAll, fields.Everything())
	epstore, epcontroller := cache.NewInformer(
		epListWatch,
		&v1.Endpoints{},
		time.Minute*10,
		cache.ResourceEventHandlerFuncs{
			AddFunc: clbctr.onEpAdd,
			DeleteFunc: clbctr.onEpDel,
			UpdateFunc: clbctr.onEpUpdate,
		},
	)	
	clbctr.epController = epcontroller
	clbctr.epstore = epstore
	
	return clbctr, nil
}

func (c *CLBController)Run(ctx <-chan struct{}) {
	//Run CLB Controller
	glog.V(2).Infof("CLB Controller starting...")
	go c.clbController.Run(ctx)
	wait.Poll(time.Second, 5*time.Minute, func() (bool, error) {
		return c.clbController.HasSynced(), nil
	})
	if !c.clbController.HasSynced() {
		glog.Errorf("clb informer initial sync timeout")
		os.Exit(1)
	}
	
	//Run Endpoints Controller
	glog.V(2).Infof("Endpoint Controller starting...")
	go c.epController.Run(ctx)
	wait.Poll(time.Second, 5*time.Minute, func() (bool, error) {
		return c.epController.HasSynced(), nil
	})
	if !c.epController.HasSynced() {
		glog.Errorf("endpoint informer initial sync timeout")
		os.Exit(1)
	}	
}

func (c *CLBController)onClbAdd(obj interface{}) {
	clb := obj.(*crdv1.ClassicLoadBalance)
	namespace := clb.Namespace
	glog.V(3).Infof("Add-CLB[%s]: %#v", namespace, clb)
	
	for _, store := range c.epstore.List() {
		glog.V(5).Infof("Iterator epstore: %#v", store)
	}
	for _, storekey := range c.epstore.ListKeys() {
		glog.V(5).Infof("Iterator epstoreKey: %s", storekey)
	}	
	
	var vip string
	var err error
	// TODO:  Get Vip from openstack
	if clb.Spec.IP != "" {
		vip = clb.Spec.IP
	}
	port := clb.Spec.Port
	protocol := clb.Spec.Protocol
	lbname, err := c.driver.CreateLb(namespace, vip, port, protocol)
	if err != nil {
		glog.Errorf("CreateLb failed : %v", err)
		return
	} else {
		glog.V(2).Infof("Create LB %s succeced.", lbname)
	}
	for _, backend := range clb.Spec.Backends {
		weight := backend.Weight
		if weight <= 0 {
			weight = 1
		}
		
		svcgrp, err := c.driver.CreateSvcGroup(namespace, 
			backend.ServiceName)
		if err != nil {
			glog.Errorf("CreateSvcGrp %s failed : %v", backend.ServiceName, err)
		}
		err = c.driver.BindSvcGroupLb(svcgrp, lbname)
		if err != nil {
			glog.Errorf("BindSvcGroup %s to Lb %s failed : %v", backend.ServiceName, lbname, err)
			return			
		}
		
		svckey := namespace + "/" + backend.ServiceName
		eps, exists, err := c.epstore.GetByKey(svckey)
		
		if exists && (err == nil) {
			epss := eps.(*v1.Endpoints)
			if len(epss.Subsets) < 1 {
				glog.V(3).Infof("[%s]Get Empty Eps: %#v", svckey, epss.Subsets)
				continue
			}
			glog.V(4).Infof("[%s]Get Eps: %#v", svckey, epss.Subsets[0])
			for _, epaddr := range epss.Subsets[0].Addresses {
				ip := epaddr.IP
				for _, epport := range epss.Subsets[0].Ports {
					port := epport.Port
					//protocol := string(epport.Protocol)
					//svcname, err := c.driver.CreateSvc(namespace, ip, port, protocol)
					srv, err := c.driver.CreateServer(namespace, ip)
					if err != nil {
						glog.Errorf("Create server %s failed: %v", srv, err)
					}
					//err = c.driver.BindSvcToLb(svcname, lbname, weight)
					c.driver.BindServerToGroup(srv, svcgrp, port, weight)
					if err != nil {
						glog.Errorf("Bind svc to lb failed: %v", err)
					}
				}
			}
		}		
	}
	
	clb.Status.State = "Available"
	clbclient := crdclient.ClbClient(c.crdClient, c.crdScheme, namespace)
	glog.V(5).Infof("Update ClaasicLoadBalance Status: %+v", clb)
	_, err = clbclient.Update(clb, clb.Name)
	if err != nil {
		glog.Errorf("Update loadbalance failed: %v", err)
	}
}

func (c *CLBController)onClbUpdate(oldObj, newObj interface{}) {
	glog.V(3).Infof("Update-CLB: %+v -> %+v", oldObj, newObj)
}

func (c *CLBController)onClbDel(obj interface{}) {
	glog.V(3).Infof("Del-CLB: %#v", obj)
	clb := obj.(*crdv1.ClassicLoadBalance)
	
	c.driver.DeleteLb(clb.Namespace, clb.Spec.IP, clb.Spec.Port, clb.Spec.Protocol)
}

func (c *CLBController)onEpAdd(obj interface{}) {
	glog.V(3).Infof("Add-Ep: %v", obj)
}

func (c *CLBController)onEpUpdate(oldObj, newObj interface{}) {
	glog.V(4).Infof("Update-Ep: %v -> %v", oldObj, newObj)
	if !reflect.DeepEqual(oldObj, newObj) {
		oldclb := oldObj.(*v1.Endpoints)
		newclb := newObj.(*v1.Endpoints)
		glog.V(4).Infof("Update-Diff Ep: %s-> %s", oldclb.Name , newclb.Name )
	}
}

func (c *CLBController)onEpDel(obj interface{}) {
	glog.V(3).Infof("Del-Ep: %v", obj)
}