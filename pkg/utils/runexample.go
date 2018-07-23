package utils

import (
	"github.com/golang/glog"
		
	"github.com/sak0/lb-operator/pkg/client"
	crdv1 "github.com/sak0/lb-operator/pkg/apis/loadbalance/v1"
	
	"k8s.io/apimachinery/pkg/runtime"	
	apiextcs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

func RunAlbExample(crdClient *rest.RESTClient, crdScheme *runtime.Scheme){
	// Create a CRD client interface
	albclient := client.AlbClient(crdClient, crdScheme, "default")

	// Test: Create a new AppLoadBalance object and write to k8s
	alb := &crdv1.AppLoadBalance{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:   "lb123",
			Labels: map[string]string{"mylabel": "test"},
		},
		Spec: crdv1.AppLoadBalanceSpec{
			IP: "10.0.12.168",
			Port: "80",
			Rules: []crdv1.AppLoadBalanceRule{
				crdv1.AppLoadBalanceRule{
					Host : "ingress.yonghui.cn",
					Paths : []crdv1.AppLoadBalancePath{
						crdv1.AppLoadBalancePath{
							Path : "/demo",
							Backend : crdv1.AppLoadBalanceBackend{
								ServiceName : "demoSvc",
								ServicePort : 80,
							},
						},
					},
				},
			},
		},
		Status: crdv1.AppLoadBalanceStatus{
			State:   "created",
			Message: "Created, not processed yet",
		},
	}

	result, err := albclient.Create(alb)
	if err == nil {
		glog.V(3).Infof("CREATED: %#v", result)
	} else if apierrors.IsAlreadyExists(err) {
		glog.V(3).Infof("ALREADY EXISTS: %#v", result)
	} else {
		panic(err)
	}

	// List all AppLoadBalance objects
	items, err := albclient.List(meta_v1.ListOptions{})
	if err != nil {
		panic(err)
	}
	glog.V(3).Infof("List: \n%v", items)
}

func RunClbExample(crdClient *rest.RESTClient, crdScheme *runtime.Scheme){
	// Create a CRD client interface
	clbclient := client.ClbClient(crdClient, crdScheme, "default")

	// Test: Create a new AppLoadBalance object and write to k8s
	clb := &crdv1.ClassicLoadBalance{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:   "clb123",
			Labels: map[string]string{"mylabel": "test"},
		},
		Spec: crdv1.ClassicLoadBalanceSpec{
			IP: "10.0.12.168",
			Port: "80",
			Backends: []crdv1.ClassicLoadBalanceBackend{
				crdv1.ClassicLoadBalanceBackend{
					ServiceName : "demosvc",
					//ServicePort : 80,					
				},
			},
		},
		Status: crdv1.ClassicLoadBalanceStatus{
			State:   "created",
			Message: "Created, not processed yet",
		},
	}
	
	resultclb, err := clbclient.Create(clb)
	if err == nil {
		glog.V(3).Infof("CREATED: %#v", resultclb)
	} else if apierrors.IsAlreadyExists(err) {
		glog.V(3).Infof("ALREADY EXISTS: %#v", resultclb)
	} else {
		panic(err)
	}	
}

func InitAllCRD(extClient *apiextcs.Clientset)error{
	// note: if the CRD exist our CreateCRD function is set to exit without an error
	err := crdv1.CreateALBCRD(extClient)
	if err != nil {
		return err
	}
	err = crdv1.CreateCLBCRD(extClient)
	if err != nil {
		return err
	}

	// Wait for the CRD to be created before we use it (only needed if its a new one)
	if err := WaitCRDReady(extClient, crdv1.FullALBName); err != nil {
		glog.Errorf("Wait ALB Crd Create failed: %v", err)
		return err
	}
	if err := WaitCRDReady(extClient, crdv1.FullCLBName); err != nil {
		glog.Errorf("Wait CLB Crd Create failed: %v", err)
		return err
	}
	
	return nil
}