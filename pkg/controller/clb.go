package controller

import (
	"fmt"
	"time"
	"os"
	"reflect"
	"strconv"
	"strings"
	
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
	"github.com/sak0/lb-operator/pkg/utils"
)

type CLBController struct {
	crdClient		*rest.RESTClient
	crdScheme		*runtime.Scheme
	client			kubernetes.Interface
	
	clbController	cache.Controller
	epController	cache.Controller
	
	clbstore     	cache.Store
	epstore			cache.Store
	
	clbSvcRef		map[string]map[string]int
	
	driver			driver.LbProvider			
}

func NewCLBController(client kubernetes.Interface, crdClient *rest.RESTClient, 
					crdScheme *runtime.Scheme)(*CLBController, error) {
	clbctr := &CLBController{
		crdClient 	: crdClient,
		crdScheme 	: crdScheme,
		client		: client,
		clbSvcRef   : make(map[string]map[string]int),
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
}

func (c *CLBController)onClbAdd(obj interface{}) {
	clb := obj.(*crdv1.ClassicLoadBalance)
	defer clbTotal.Inc()
	
	namespace := clb.Namespace
	glog.V(3).Infof("Add-CLB[%s]: %#v", namespace, clb)
	
	for _, store := range c.epstore.List() {
		glog.V(5).Infof("Iterator epstore: %#v", store)
	}
	for _, storekey := range c.epstore.ListKeys() {
		glog.V(5).Infof("Iterator epstoreKey: %s", storekey)
	}	
		
	vip, err := c.ensureVip(clb)
	if err != nil {
		c.updateError(err.Error(), clb)
		return		
	}
	clb.Spec.IP = vip
		
	port := clb.Spec.Port
	protocol := clb.Spec.Protocol
	//cause k8s service unsupport http
	if protocol != "HTTP" && protocol != "TCP" && protocol != "UDP" {
		c.updateError(fmt.Sprintf("protocol %s is unsupported.", protocol), clb)
		return
	} else if protocol == "HTTP"{
		protocol = "TCP"
	}
	
	lbname, err := c.driver.CreateLb(namespace, vip, port, protocol)
	if err != nil {
		glog.Errorf("CreateLb failed : %v", err)
		return
	} else {
		glog.V(2).Infof("Create LB %s succeced.", lbname)
	}
	
	for _, backend := range clb.Spec.Backends {
		c.addBackendToCLB(clb, backend)	
	}
	
	glog.V(2).Infof("Update ClaasicLoadBalance Status: %+v", clb)
	c.updateAvailable("", clb)
}

func (c *CLBController)onClbUpdate(oldObj, newObj interface{}) {
	glog.V(3).Infof("Update-CLB: %+v -> %+v", oldObj, newObj)
	
	if !reflect.DeepEqual(oldObj, newObj) {
		newClb := newObj.(*crdv1.ClassicLoadBalance)
		oldClb := oldObj.(*crdv1.ClassicLoadBalance)
				
		backendsNew := utils.GetBackendMap(newClb)
		backendsOld := utils.GetBackendMap(oldClb)
		glog.V(2).Infof("backendsNew: %v", backendsNew)
		glog.V(2).Infof("backendsOld: %v", backendsOld)
		if !reflect.DeepEqual(backendsNew, backendsOld) {
			glog.V(2).Infof("Need update CLB configurations.")
			c.updateClb(newClb, backendsNew, backendsOld)
		}					
	}
}

func (c *CLBController)onClbDel(obj interface{}) {
	glog.V(3).Infof("Del-CLB: %#v", obj)
	defer clbTotal.Dec()
	clb := obj.(*crdv1.ClassicLoadBalance)
		
	lbName := utils.GenerateLbNameCLB(clb.Namespace, clb.Spec.IP, clb.Spec.Port, clb.Spec.Protocol)
	for _, backend := range clb.Spec.Backends {
		lbNameMap := c.clbSvcRef[backend.ServiceName]
		delete(lbNameMap, lbName)
		
		c.removeBackendFromCLB(clb, backend)
	}
	
	protocol := clb.Spec.Protocol
	if protocol == "HTTP" {
		protocol = "TCP"
	}
	c.driver.DeleteLb(clb.Namespace, clb.Spec.IP, clb.Spec.Port, protocol)
	utils.ReleaseIpAddr(clb.Namespace, clb.Spec.IP)	
}

func (c *CLBController)removeBackendFromCLB(clb *crdv1.ClassicLoadBalance, backend crdv1.ClassicLoadBalanceBackend){
	lbName := utils.GenerateLbNameCLB(clb.Namespace, clb.Spec.IP, 
			clb.Spec.Port, clb.Spec.Protocol)	

	svcGroupName := utils.GenerateSvcGroupNameCLB(clb.Namespace, clb.Name, backend.ServiceName, backend.ServicePort)
	
	err := c.driver.UnBindSvcGroupLb(svcGroupName, lbName)
	if err != nil {
		glog.Errorf("UnindSvcGroup %s to Lb %s failed : %v", svcGroupName, lbName, err)		
	}
	err = c.driver.DeleteSvcGroup(svcGroupName)
	if err != nil {
		glog.Errorf("DeleteSvcGroup %s failed : %v", svcGroupName, err)
	}	
}

func (c *CLBController)addBackendToCLB(clb *crdv1.ClassicLoadBalance, backend crdv1.ClassicLoadBalanceBackend){
//	if lbNameMap, ok := c.clbSvcRef[backend.ServiceName]; !ok {
//		newlbNameMap := make(map[string]int)
//		newlbNameMap[lbname] = 1
//		c.clbSvcRef[backend.ServiceName] = newlbNameMap
//	} else {
//		//TODO adapt to one lb ref the service more than 1 times
//		lbNameMap[lbname] = 1
//		c.clbSvcRef[backend.ServiceName] = lbNameMap
//	}
	lbName := utils.GenerateLbNameCLB(clb.Namespace, clb.Spec.IP, 
			clb.Spec.Port, clb.Spec.Protocol)	

	weight := backend.Weight
	if weight <= 0 {
		weight = 1
	}
	svcGroupName := utils.GenerateSvcGroupNameCLB(clb.Namespace, clb.Name, backend.ServiceName, backend.ServicePort)
	err := c.driver.CreateSvcGroup(svcGroupName)
	if err != nil {
		glog.Errorf("CreateSvcGrp %s failed : %v", backend.ServiceName, err)
	}
	
	err = c.driver.BindSvcGroupLb(svcGroupName, lbName)
	if err != nil {
		glog.Errorf("BindSvcGroup %s to Lb %s failed : %v", backend.ServiceName, clb.Name, err)		
	}
	
	svckey := clb.Namespace + "/" + backend.ServiceName
	eps, exists, err := c.epstore.GetByKey(svckey)
	
	if exists && (err == nil) {
		epss := eps.(*v1.Endpoints)
		if len(epss.Subsets) < 1 {
			glog.V(3).Infof("[%s]Get Empty Eps: %#v", svckey, epss.Subsets)
			return
		}
		glog.V(4).Infof("[%s]Get Eps: %#v", svckey, epss.Subsets[0])
		for _, epaddr := range epss.Subsets[0].Addresses {
			ip := epaddr.IP
			for _, epport := range epss.Subsets[0].Ports {
				port := int(epport.Port)
				if strconv.Itoa(port) != backend.ServicePort {
					glog.V(2).Infof("Ignore service port %d, backendPort = %s\n", port, backend.ServicePort)
					continue
				}

				//protocol := string(epport.Protocol)
				//svcname, err := c.driver.CreateSvc(namespace, ip, port, protocol)
				srv, err := c.driver.CreateServer(clb.Namespace, ip)
				if err != nil {
					glog.Errorf("Create server %s failed: %v", srv, err)
				}
				//err = c.driver.BindSvcToLb(svcname, lbname, weight)
				c.driver.BindServerToGroup(srv, svcGroupName, port, weight)
				if err != nil {
					glog.Errorf("Bind svc to lb failed: %v", err)
				}
			}
		}
	}
}

func (c *CLBController)updateClb(newClb *crdv1.ClassicLoadBalance, 
	backendsNew map[crdv1.ClassicLoadBalanceBackend]int, 
	backendsOld map[crdv1.ClassicLoadBalanceBackend]int) {
	
	lbName := utils.GenerateLbNameCLB(newClb.Namespace, newClb.Spec.IP, 
			newClb.Spec.Port, newClb.Spec.Protocol)	
		
	for backendNew, _ := range backendsNew {
		if _, ok := backendsOld[backendNew]; !ok {
			glog.V(2).Infof("CLB Update: need add backend %v to %s", backendNew, lbName)
			c.addBackendToCLB(newClb, backendNew)
		}
	}
	
	for backendOld, _ := range backendsOld {
		if _, ok := backendsNew[backendOld]; !ok {
			glog.V(2).Infof("CLB Update: need remove backend %v from %s", backendOld, lbName)
			c.removeBackendFromCLB(newClb, backendOld)
			
			lbNameMap := c.clbSvcRef[backendOld.ServiceName]
			delete(lbNameMap, lbName)
		}
	}
}

func (c *CLBController)ensureVip(clb *crdv1.ClassicLoadBalance)(string, error){
	if clb.Status.State == crdv1.CLBSTATUSAVAILABLE {
		return clb.Spec.IP, nil
	}
	
	var vip string
	var err error
	if clb.Spec.IP != "" {
		vip = clb.Spec.IP
		err = utils.CreatePortFromIp(clb.Namespace, vip, clb.Spec.Subnet)
		if err != nil {
			glog.Errorf("Create port from ip failed: %v", err)
			return vip, err			
		}
	} else {
		vip, err = utils.AllocIpAddrFromSubnet(clb.Namespace, clb.Spec.Subnet)
		if err != nil {
			glog.Errorf("Alloc ip failed: %v", err)
			return "", err
		} else {
			glog.V(2).Infof("CreateCLB with vip: %s", vip)	
		}
	}
	
	return vip, nil
}		

func (c *CLBController)onEpAdd(obj interface{}) {
	glog.V(3).Infof("Add-Ep: %v", obj)
}

func (c *CLBController)createAndBindServer(namespace string, ipstr string, groupName string)error{
	ipstrArray := strings.Split(ipstr, ":")
	ip := ipstrArray[0]
	port, _  := strconv.Atoi(ipstrArray[1])
	srv, err := c.driver.CreateServer(namespace, ip)
	if err != nil {
		glog.Errorf("Create server %s failed: %v", srv, err)
	}
	
	//TODO Get weight
	err = c.driver.BindServerToGroup(srv, groupName, port, 1)
	if err != nil {
		glog.Errorf("Bind svc to lb failed: %v", err)
	}
	return nil	
}

func (c *CLBController)deleteAndUnBindServer(namespace string, ipstr string, groupName string)error {
	ipstrArray := strings.Split(ipstr, ":")
	ip := ipstrArray[0]
	port, _  := strconv.Atoi(ipstrArray[1])
	serverName := utils.GenerateServerNameCLB(namespace, ip)
	
	err := c.driver.UnBindServerFromGroup(serverName, groupName, port)
	if err != nil {
		glog.Errorf("UnBind svc from svcgrp failed: %v", err)
	}
	
	return nil	
}

//func (c *CLBController)updateEndpoints(namespace string, epName string, 
//	epsNew map[string]int, epsOld map[string]int){
//	groupName := utils.GenerateSvcGroupNameCLB(namespace, epName)
//	for newips, _ := range epsNew {
//		if _, ok := epsOld[newips]; !ok {
//			glog.V(2).Infof("Need add Server %s on %s", newips, groupName)
//			_ = c.createAndBindServer(namespace, newips, groupName)
//		}
//	}
//	for oldips, _ := range epsOld {
//		if _, ok := epsNew[oldips]; !ok {
//			glog.V(2).Infof("Need del Server %s on %s", oldips, groupName)
//			_ = c.deleteAndUnBindServer(namespace, oldips, groupName)
//		}
//	}
//}
	
func (c *CLBController)updateEndpoints(svcName string, lbs []string, epsNew, epsOld map[string]int) {
	for newips, _ := range epsNew {
		if _, ok := epsOld[newips]; !ok {
			glog.V(2).Infof("Prepare add Server %s lbs %v", newips, lbs)
			for _, lb := range lbs {
				// TODO: move string join/split logic to utils
				namespace := strings.Split(lb, ":")[0]
				lbname := strings.Split(lb, ":")[1]
				lbport := strings.Split(lb, ":")[2]
				port := strings.Split(newips, ":")[1]
				if lbport == port {
					groupName := utils.GenerateSvcGroupNameCLB(namespace, lbname, svcName, port)
					glog.V(2).Infof("Need add Server %s to group %s", newips, groupName)
					_ = c.createAndBindServer(namespace, newips, groupName)
				}
			}
		}
	}
	for oldips, _ := range epsOld {
		if _, ok := epsNew[oldips]; !ok {
			glog.V(2).Infof("Prepare del Server %s from lbs %s", oldips, lbs)
			for _, lb := range lbs {
				// TODO: move string join/split logic to utils
				namespace := strings.Split(lb, ":")[0]
				lbname := strings.Split(lb, ":")[1]
				lbport := strings.Split(lb, ":")[2]
				port := strings.Split(oldips, ":")[1]
				if lbport == port {
					groupName := utils.GenerateSvcGroupNameCLB(namespace, lbname, svcName, port)
					glog.V(2).Infof("Need del Server %s from group %s", oldips, groupName)
					_ = c.deleteAndUnBindServer(namespace, oldips, groupName)
				}
			}			
		}
	}		
}	

func (c *CLBController)onEpUpdate(oldObj, newObj interface{}) {
	glog.V(4).Infof("Update-Ep: %v -> %v", oldObj, newObj)
	glog.V(4).Infof("clbSvcRef: %v", c.clbSvcRef)
	if !reflect.DeepEqual(oldObj, newObj) {
		oldep := oldObj.(*v1.Endpoints)
		newep := newObj.(*v1.Endpoints)
				
		LbSvcMap := utils.GetLbSvcMap(c.clbstore)
		glog.V(4).Infof("LbSvcMap: %+v\n", LbSvcMap)
		lbs, ok := LbSvcMap[newep.Name]
		if !ok {
			glog.V(4).Infof("Ignore service %s update.", newep.Name)
			return
		}
		
		epsNew := utils.GetEndpointMap(newep)
		epsOld := utils.GetEndpointMap(oldep)
		if !reflect.DeepEqual(epsNew, epsOld) {
			glog.V(2).Infof("Need Update for service %s lbs %+v\n", oldep.Name, lbs)
			c.updateEndpoints(newep.Name, lbs, epsNew, epsOld)
		}		

//		glog.V(2).Infof("Update-Diff Ep: %s-> %s", oldep.Name , newep.Name)
//		
//		_, found := c.clbSvcRef[newep.Name]
//		if found {
//			glog.V(2).Infof("Ep %s have refcount with lb-operator", newep.Name)
//			epsNew := utils.GetEndpointMap(newep)
//			epsOld := utils.GetEndpointMap(oldep)
//			glog.V(2).Infof("NewEps: %v", epsNew)
//			glog.V(2).Infof("OldEps: %v", epsOld)
//			if !reflect.DeepEqual(epsNew, epsOld) {
//				glog.V(2).Infof("Need update clb configurations.")
//				c.updateEndpoints(newep.Namespace, newep.Name, epsNew, epsOld)
//			}
//		}
	}
}

func (c *CLBController)onEpDel(obj interface{}) {
	glog.V(3).Infof("Del-Ep: %v", obj)
}

func (c *CLBController)updateAvailable(msg string, clb *crdv1.ClassicLoadBalance) {
	clb.Status.State = crdv1.CLBSTATUSAVAILABLE
	clb.Status.Message = msg
	clbclient := crdclient.ClbClient(c.crdClient, c.crdScheme, clb.Namespace)
	_, _ = clbclient.Update(clb, clb.Name)
}

func (c *CLBController)updateError(msg string, clb *crdv1.ClassicLoadBalance) {
	clb.Status.State = crdv1.CLBSTATUSERROR
	clb.Status.Message = msg
	clbclient := crdclient.ClbClient(c.crdClient, c.crdScheme, clb.Namespace)
	_, _ = clbclient.Update(clb, clb.Name)
}