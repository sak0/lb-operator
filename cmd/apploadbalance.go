package main

import (
	"time"
	"flag"
	"os"
	"os/signal"
	"syscall"
		
	"github.com/golang/glog"

	"github.com/sak0/lb-operator/pkg/client"
	"github.com/sak0/lb-operator/pkg/controller"
	"github.com/sak0/lb-operator/pkg/utils"
	crdv1 "github.com/sak0/lb-operator/pkg/apis/loadbalance/v1"
	
	clientset "k8s.io/client-go/kubernetes"
	apiextcs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/rest"
)

func GetClientConfig(kubeconfig string) (*rest.Config, error) {
	/*if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}*/
	return rest.InClusterConfig()
}

func main() {
	kubeconf := flag.String("kubeconf", "admin.conf", "Path to a kube config. Only required if out-of-cluster.")
	runTest := flag.Bool("runtest", false, "If create test resource.")
	flag.Parse()

	config, err := GetClientConfig(*kubeconf)
	if err != nil {
		panic(err.Error())
	}

	// create extclient and create our CRD, this only need to run once
	extClient, err := apiextcs.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	
	kubeClient, err := clientset.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	// note: if the CRD exist our CreateCRD function is set to exit without an error
	err = crdv1.CreateALBCRD(extClient)
	if err != nil {
		panic(err)
	}
	err = crdv1.CreateCLBCRD(extClient)
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
	
	if *runTest {
		glog.V(2).Infof("Creating test resource...")
		utils.RunAlbExample(crdcs, scheme)
		utils.RunClbExample(crdcs, scheme)
	}

	// AppLoadBalance Controller
	// Watch for changes in AppLoadBalance objects and fire Add, Delete, Update callbacks
	stopCh := make(chan struct{})
	albctr, _ := controller.NewALBController(kubeClient, crdcs, scheme)
	clbctr, _ := controller.NewCLBController(kubeClient, crdcs, scheme)
	go albctr.Run(stopCh)
	go clbctr.Run(stopCh)


	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	glog.V(2).Infof("signal.Notify ready..")
	<-c
	close(stopCh)
	glog.V(2).Infof("Bye bye...")
}
