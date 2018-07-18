package main

import (
	"time"
	"github.com/golang/glog"

	"github.com/sak0/lb-operator/pkg/client"
	crdv1 "github.com/sak0/lb-operator/pkg/apis/loadbalance/v1"

	apiextcs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	//"k8s.io/client-go/tools/clientcmd"
	"flag"
)

func GetClientConfig(kubeconfig string) (*rest.Config, error) {
	/*if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}*/
	return rest.InClusterConfig()
}

func main() {
	kubeconf := flag.String("kubeconf", "admin.conf", "Path to a kube config. Only required if out-of-cluster.")
	flag.Parse()

	config, err := GetClientConfig(*kubeconf)
	if err != nil {
		panic(err.Error())
	}

	// create clientset and create our CRD, this only need to run once
	clientset, err := apiextcs.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	// note: if the CRD exist our CreateCRD function is set to exit without an error
	err = crdv1.CreateALBCRD(clientset)
	if err != nil {
		panic(err)
	}

	// Wait for the CRD to be created before we use it (only needed if its a new one)
	time.Sleep(3 * time.Second)

	// Create a new clientset which include our CRD schema
	crdcs, scheme, err := client.NewClient(config)
	if err != nil {
		panic(err)
	}

	// Create a CRD client interface
	albclient := client.AlbClient(crdcs, scheme, "default")

	// Test: Create a new AppLoadBalance object and write to k8s
	lb := &crdv1.AppLoadBalance{
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

	result, err := albclient.Create(lb)
	if err == nil {
		glog.V(2).Infof("CREATED: %#v", result)
	} else if apierrors.IsAlreadyExists(err) {
		glog.V(2).Infof("ALREADY EXISTS: %#v", result)
	} else {
		panic(err)
	}

	// List all AppLoadBalance objects
	items, err := albclient.List(meta_v1.ListOptions{})
	if err != nil {
		panic(err)
	}
	glog.V(3).Infof("List: \n%v", items)

	// AppLoadBalance Controller
	// Watch for changes in AppLoadBalance objects and fire Add, Delete, Update callbacks
	_, controller := cache.NewInformer(
		albclient.NewListWatch(),
		&crdv1.AppLoadBalance{},
		time.Minute*10,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				glog.V(3).Infof("Add: %v", obj)
			},
			DeleteFunc: func(obj interface{}) {
				glog.V(3).Infof("Delete: %v", obj)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				glog.V(3).Infof("Update old: %v \n New: %v", oldObj, newObj)
			},
		},
	)

	stop := make(chan struct{})
	go controller.Run(stop)

	select {}
}
