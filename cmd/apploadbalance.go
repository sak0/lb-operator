package main

import (
	"flag"
	"log"
	"net"
	"net/http"	
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/sak0/lb-operator/pkg/controller"
	"github.com/sak0/lb-operator/pkg/utils"
)

const (
	healthzPath = "/healthz"
)

var (
	kubeConf	= flag.String("kubeconf", "admin.conf", "Path to a kube config. Only required if out-of-cluster.")
	runTest		= flag.Bool("runtest", false, "If create test resource.")
	createCrd	= flag.Bool("createCrd", true, "If create crd.")
	
	metricsPath	= flag.String("metrics-path", "/metrics", "metrcis url path.")
	metricsPort	= flag.Int("port", 8888, "metrics listen port.")
)

func main() {
	flag.Parse()
	
	// Get all clients
	kubeClient, extClient, crdcs, scheme, err := utils.CreateClients(*kubeConf)
	
	if err != nil {
		panic(err.Error())
	}

	//Init CRD Object if needed
	if *createCrd == true {
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


	//Create exporter for prometheus
	http.Handle(*metricsPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
			<head><title>LoadBalance Controller</title></head>
			<body>
			<h1>Hello LB</h1>
			<p><a href='` + *metricsPath + `'>Metrics</a></p>
			</body>
			</html>`))
	})
	http.HandleFunc(healthzPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})
	listenAddress := net.JoinHostPort("0.0.0.0", strconv.Itoa(*metricsPort))
	go log.Fatal(http.ListenAndServe(listenAddress, nil))	
	

	//Catch signal for exit.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	glog.V(2).Infof("signal.Notify ready..")
	<-c
	close(stopCh)
	glog.V(2).Infof("Bye bye...")
}
