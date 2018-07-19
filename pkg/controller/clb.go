package controller

import (
	"time"
	"os"
	
	"github.com/golang/glog"
	
	"k8s.io/client-go/kubernetes"	
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	
	crdv1 "github.com/sak0/lb-operator/pkg/apis/loadbalance/v1"
)

type CLBController struct {
	crdClient		*rest.RESTClient
	crdScheme		*runtime.Scheme
	client			kubernetes.Interface
	
	clbController	cache.Controller
}

func NewCLBController(client kubernetes.Interface, crdClient *rest.RESTClient, 
					crdScheme *runtime.Scheme)(*CLBController, error) {
	clbctr := &CLBController{
		crdClient 	: crdClient,
		crdScheme 	: crdScheme,
		client		: client,
	}
	
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
}

func (c *CLBController)onClbAdd(obj interface{}) {
	glog.V(3).Infof("Add-CLB: %v", obj)
}

func (c *CLBController)onClbUpdate(oldObj, newObj interface{}) {
	glog.V(3).Infof("Update-CLB: %v -> %v", oldObj, newObj)
}

func (c *CLBController)onClbDel(obj interface{}) {
	glog.V(3).Infof("Del-CLB: %v", obj)
}