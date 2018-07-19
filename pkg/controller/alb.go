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

type ALBController struct {
	crdClient		*rest.RESTClient
	crdScheme		*runtime.Scheme
	client			kubernetes.Interface
	
	albController	cache.Controller
}

func NewALBController(client kubernetes.Interface, crdClient *rest.RESTClient, 
					crdScheme *runtime.Scheme)(*ALBController, error) {
	albctr := &ALBController{
		crdClient 	: crdClient,
		crdScheme 	: crdScheme,
		client		: client,
	}
	
	albListWatch := cache.NewListWatchFromClient(albctr.crdClient, 
		crdv1.ALBPlural, meta_v1.NamespaceAll, fields.Everything())
	
	_, albcontroller := cache.NewInformer(
		albListWatch,
		&crdv1.AppLoadBalance{},
		time.Minute*10,
		cache.ResourceEventHandlerFuncs{
			AddFunc: albctr.onAlbAdd,
			DeleteFunc: albctr.onAlbDel,
			UpdateFunc: albctr.onAlbUpdate,
		},
	)
	albctr.albController = albcontroller
	
	return albctr, nil
}

func (c *ALBController)Run(ctx <-chan struct{}) {
	glog.V(2).Infof("ALB Controller starting...")
	go c.albController.Run(ctx)
	wait.Poll(time.Second, 5*time.Minute, func() (bool, error) {
		return c.albController.HasSynced(), nil
	})
	if !c.albController.HasSynced() {
		glog.Errorf("alb informer initial sync timeout")
		os.Exit(1)
	}
}

func (c *ALBController)onAlbAdd(obj interface{}) {
	glog.V(3).Infof("Add-ALB: %v", obj)
}

func (c *ALBController)onAlbUpdate(oldObj, newObj interface{}) {
	glog.V(3).Infof("Update-ALB: %v -> %v", oldObj, newObj)
}

func (c *ALBController)onAlbDel(obj interface{}) {
	glog.V(3).Infof("Del-ALB: %v", obj)
}