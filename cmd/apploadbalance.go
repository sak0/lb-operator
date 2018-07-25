package main

import (
	"flag"
	"net"
	"net/http"	
	"strconv"
	"time"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"

	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"	

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
	
	//TODO read from env.
	electionName		= flag.String("name", "lb-operator", "electionName for this instance.")
	electionId			= flag.String("id", "host123", "electionId for this instance.")
	electionNamespace	= flag.String("namespace", "default", "election resource's Namespace.")
)

func run(stopCh <-chan struct{}){
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
	/*// Create some test resources if needed
	if *runTest {
		glog.V(2).Infof("Creating test resource...")
		utils.RunAlbExample(crdcs, scheme)
		utils.RunClbExample(crdcs, scheme)
	}*/		

	// Run controllers
	// Watch for changes in AppLoadBalance and ClassicLoadBalance objects and fire Add, Delete, Update callbacks
	//stopCh := make(chan struct{})
	albctr, _ := controller.NewALBController(kubeClient, crdcs, scheme)
	clbctr, _ := controller.NewCLBController(kubeClient, crdcs, scheme)
	go albctr.Run(stopCh)
	go clbctr.Run(stopCh)
}


func main() {
	flag.Parse()	
	
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
	go http.ListenAndServe(listenAddress, nil)
	
	kubeclient := utils.MustNewKubeClient()
	glog.V(2).Infof("%v", kubeclient)

	rl, err := resourcelock.New(resourcelock.EndpointsResourceLock,
		*electionNamespace,
		"lb-operator",
		kubeclient.Core(),
		resourcelock.ResourceLockConfig{
			Identity:      *electionId,
			EventRecorder: createRecorder(kubeclient, *electionName, *electionNamespace),
		})
	if err != nil {
		glog.Fatalf("error creating lock: %v", err)
		panic(err)
	}

	leaderelection.RunOrDie(leaderelection.LeaderElectionConfig{
		Lock:          rl,
		LeaseDuration: 15 * time.Second,
		RenewDeadline: 10 * time.Second,
		RetryPeriod:   2 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: run,
			OnStoppedLeading: func() {
				glog.Fatalf("leader election lost")
			},
		},
	})	
	
	
	/*stopCh := make(<-chan struct{})
	run(stopCh)
	//Catch signal for exit.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	glog.V(2).Infof("signal.Notify ready..")
	<-c
	//close(stopCh)
	glog.V(2).Infof("Bye bye...")*/
}

func createRecorder(kubecli kubernetes.Interface, name, namespace string) record.EventRecorder {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&v1core.EventSinkImpl{Interface: v1core.New(kubecli.Core().RESTClient()).Events(namespace)})
	return eventBroadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: name})
}
