package controller

import (
	"time"
	"os"
	
	"github.com/golang/glog"
	
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"	
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	
	crdv1 "github.com/sak0/lb-operator/pkg/apis/loadbalance/v1"
	driver "github.com/sak0/lb-operator/pkg/drivers"
)

type CLBController struct {
	crdClient		*rest.RESTClient
	crdScheme		*runtime.Scheme
	client			kubernetes.Interface
	
	clbController	cache.Controller
	epController	cache.Controller
	
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
	
	_, clbcontroller := cache.NewInformer(
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
	
	//Construction Endpoint Informer
	epListWatch := cache.NewListWatchFromClient(client.Core().RESTClient(), 
		"endpoints", meta_v1.NamespaceAll, fields.Everything())
	_, epcontroller := cache.NewInformer(
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
	
	return clbctr, nil
}

func (c *CLBController)Run(ctx <-chan struct{}) {
	glog.V(2).Infof("CLB Controller starting...")
	go c.clbController.Run(ctx)
	wait.Poll(time.Second, 5*time.Minute, func() (bool, error) {
		return c.clbController.HasSynced(), nil
	})
	if !c.clbController.HasSynced() {
		glog.Errorf("clb informer initial sync timeout")
		os.Exit(1)
	}

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
	glog.V(3).Infof("Add-CLB: %#v", clb)
	
	var vip string
	var err error
	if clb.Spec.IP != "" {
		vip = clb.Spec.IP
	}
	port := clb.Spec.Port
	protocol := clb.Spec.Protocol
	lbname, err := c.driver.CreateLb(vip, port, protocol)
	if err != nil {
		glog.Errorf("CreateLb failed : %v", err)
		return
	}
	for _, backend := range clb.Spec.Backends {
		err = c.driver.CreateSvcGroup(backend.ServiceName)
		if err != nil {
			glog.Errorf("CreateSvcGrp %s failed : %v", backend.ServiceName, err)
			return			
		}
		err = c.driver.BindSvcGroupLb(lbname, backend.ServiceName)
		if err != nil {
			glog.Errorf("BindSvcGroup %s to Lb %s failed : %v", backend.ServiceName, lbname, err)
			return			
		}		
	}
}

func (c *CLBController)onClbUpdate(oldObj, newObj interface{}) {
	glog.V(3).Infof("Update-CLB: %v -> %v", oldObj, newObj)
}

func (c *CLBController)onClbDel(obj interface{}) {
	glog.V(3).Infof("Del-CLB: %v", obj)
}

func (c *CLBController)onEpAdd(obj interface{}) {
	glog.V(3).Infof("Add-Ep: %v", obj)
}

func (c *CLBController)onEpUpdate(oldObj, newObj interface{}) {
	glog.V(4).Infof("Update-Ep: %v -> %v", oldObj, newObj)
}

func (c *CLBController)onEpDel(obj interface{}) {
	glog.V(3).Infof("Del-Ep: %v", obj)
}