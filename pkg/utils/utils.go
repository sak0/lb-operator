package utils

import (
	"encoding/hex"
	"hash/fnv"
	"os"
	"strconv"
	"strings"
	
	"github.com/golang/glog"
	
	"github.com/sak0/lb-operator/pkg/client"

	clientset "k8s.io/client-go/kubernetes"
	apiextcs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"	
)

func getClientConfig(kubeconfig string) (*rest.Config, error) {
	/*if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}*/
	return rest.InClusterConfig()
}

func CreateClients(kubeconf string)(*clientset.Clientset, *apiextcs.Clientset, 
									*rest.RESTClient, *runtime.Scheme, error){
	config, err := getClientConfig(kubeconf)
	if err != nil {
		glog.Errorf("Get KubeConfig failed: %v", err)
		return nil, nil, nil, nil, err
	}

	// create extclient and create our CRD, this only need to run once
	extClient, err := apiextcs.NewForConfig(config)
	if err != nil {
		glog.Errorf("Get ExtApiClient failed: %v", err)
		return nil, nil, nil, nil, err
	}
	
	kubeClient, err := clientset.NewForConfig(config)
	if err != nil {
		glog.Errorf("Get KubeClient failed: %v", err)
		return nil, nil, nil, nil, err
	}
	// Create a new clientset which include our CRD schema
	crdcs, scheme, err := client.NewClient(config)
	if err != nil {
		glog.Errorf("Get CrdClient failed: %v", err)
		return nil, nil, nil, nil, err
	}
	
	return kubeClient, extClient, crdcs, scheme, nil
}
									
func hashIp()string{
	ctrlIp := os.Getenv("KUBERNETES_SERVICE_HOST")
	a := fnv.New32()
	a.Write([]byte(ctrlIp))
	return hex.EncodeToString(a.Sum(nil))
}
									
func GenerateLbNameCLB(namespace string, vip string, port string, protocol string)string {
	devHash := hashIp()
	lbName := devHash + "_CLB_" + namespace + "_" + 
			strings.Replace(vip, ".", "_", -1) + "_" + protocol + "_" + port 
	return lbName
}

func GenerateSvcNameCLB(namespace string, ip string, port int32, protocol string)string {
	devHash := hashIp()
	portstr := strconv.Itoa(int(port))
	svcName := devHash + "_CLB_" + namespace + "_" + 
			strings.Replace(ip, ".", "_", -1) + "_" + protocol + "_" + portstr
	return svcName		
}

func GenerateSvcGroupNameCLB(namespace string, svcname string)string {
	devHash := hashIp()
	gpName := devHash + "_CLB_" + namespace + "_" + svcname
	return gpName
}

func GenerateServerNameCLB(namespace string, ip string)string {
	devHash := hashIp()
	serverName := devHash + "_CLB_" + namespace + "_" + strings.Replace(ip, ".", "_", -1)
	return serverName
}

func GeneratePortNameCLB(namespace string, ip string)string {
	devHash := hashIp()
	portName := devHash + "_CLB_" + namespace + "_" + strings.Replace(ip, ".", "_", -1)
	return portName
}										