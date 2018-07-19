package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
		
	"github.com/golang/glog"

	"github.com/sak0/lb-operator/pkg/controller"
	"github.com/sak0/lb-operator/pkg/utils"
)

var (
	kubeconf	= flag.String("kubeconf", "admin.conf", "Path to a kube config. Only required if out-of-cluster.")
	runTest		= flag.Bool("runtest", false, "If create test resource.")
	createCrd	= flag.Bool("createCrd", true, "If create crd.")
)

func main() {
	flag.Parse()
	
	// Get all clients
	kubeClient, extClient, crdcs, scheme, err := utils.CreateClients(*kubeconf)
	if err != nil {
		panic(err.Error())
	}

	//Init CRD Object if needed
	if *createCrd {
		err := utils.InitAllCRD(extClient)
		if err != nil {
			panic(err.Error())
		}
	}
	
	// Create some test resources if needed
	if *runTest {
		glog.V(2).Infof("Creating test resource...")
		utils.RunAlbExample(crdcs, scheme)
		utils.RunClbExample(crdcs, scheme)
	}

	// Run controllers
	// Watch for changes in AppLoadBalance and ClassicLoadBalance objects and fire Add, Delete, Update callbacks
	stopCh := make(chan struct{})
	albctr, _ := controller.NewALBController(kubeClient, crdcs, scheme)
	clbctr, _ := controller.NewCLBController(kubeClient, crdcs, scheme)
	go albctr.Run(stopCh)
	go clbctr.Run(stopCh)


	//Catch signal for exit.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	glog.V(2).Infof("signal.Notify ready..")
	<-c
	close(stopCh)
	glog.V(2).Infof("Bye bye...")
}
