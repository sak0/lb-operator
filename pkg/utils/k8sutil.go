package utils

import (
	"fmt"
	"strconv"
	"time"	
	//crdv1 "github.com/sak0/lb-operator/pkg/apis/loadbalance/v1"
	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
)

func WaitCRDReady(clientset apiextensionsclient.Interface, crdName string) error {
	err := Retry(5*time.Second, 20, func() (bool, error) {
		crd, err := clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Get(crdName, meta_v1.GetOptions{})
		if err != nil {
			return false, err
		}
		for _, cond := range crd.Status.Conditions {
			switch cond.Type {
			case apiextensionsv1beta1.Established:
				if cond.Status == apiextensionsv1beta1.ConditionTrue {
					return true, nil
				}
			case apiextensionsv1beta1.NamesAccepted:
				if cond.Status == apiextensionsv1beta1.ConditionFalse {
					return false, fmt.Errorf("Name conflict: %v", cond.Reason)
				}
			}
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("wait CRD created failed: %v", err)
	}
	return nil
}

func GetEndpointMap(ep *v1.Endpoints)map[string]int{
	var ipmap = make(map[string]int)
	
	if len(ep.Subsets) < 1 {
		return ipmap
	}
	for _, epaddr := range ep.Subsets[0].Addresses {
		ip := epaddr.IP
		for _, epport := range ep.Subsets[0].Ports {
			port := strconv.Itoa(int(epport.Port))
			ipstr := ip + ":" + port
			ipmap[ipstr] = 1
		}
	}
	
	return ipmap
}