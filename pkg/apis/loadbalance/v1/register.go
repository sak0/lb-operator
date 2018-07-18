package v1

import (
	"reflect"
	"github.com/golang/glog"

	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextcs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	//"k8s.io/apimachinery/pkg/runtime/serializer"
	//"k8s.io/client-go/rest"
)

const (
	LBGroup			string = "loadbalance.yonghui.cn"	
	LBVersion		string = "v1"	
	
	ALBPlural		string = "apploadbalance"
	FullALBName		string = ALBPlural + "." + LBGroup

	CLBPlural		string = "classicloadbalance"
	FullCLBName		string = CLBPlural + "." + LBGroup
)

var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme	
)

// Create the CRD resource, ignore error if it already exists
func CreateALBCRD(clientset apiextcs.Interface) error {
	shotName := []string{"alb"}
	crd := &apiextv1beta1.CustomResourceDefinition{
		ObjectMeta: meta_v1.ObjectMeta{Name: FullALBName},
		Spec: apiextv1beta1.CustomResourceDefinitionSpec{
			Group:   LBGroup,
			Version: LBVersion,
			Scope:   apiextv1beta1.NamespaceScoped,
			Names:   apiextv1beta1.CustomResourceDefinitionNames{
				Plural: ALBPlural,
				Kind:   reflect.TypeOf(AppLoadBalance{}).Name(),
				ShortNames: shotName,
			},
		},
	}

	_, err := clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
	if err != nil && apierrors.IsAlreadyExists(err) {
		glog.V(2).Infof("CRD-ALB ALREADY EXISTS: %#v", crd)
		return nil
	}
	return err
}

// Create the CRD resource, ignore error if it already exists
func CreateCLBCRD(clientset apiextcs.Interface) error {
	shotName := []string{"clb"}
	crd := &apiextv1beta1.CustomResourceDefinition{
		ObjectMeta: meta_v1.ObjectMeta{Name: FullCLBName},
		Spec: apiextv1beta1.CustomResourceDefinitionSpec{
			Group:   LBGroup,
			Version: LBVersion,
			Scope:   apiextv1beta1.NamespaceScoped,
			Names:   apiextv1beta1.CustomResourceDefinitionNames{
				Plural: CLBPlural,
				Kind:   reflect.TypeOf(ClassicLoadBalance{}).Name(),
				ShortNames: shotName,
			},
		},
	}

	_, err := clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
	if err != nil && apierrors.IsAlreadyExists(err) {
		glog.V(2).Infof("CRD-CLB ALREADY EXISTS: %#v", crd)
		return nil
	}
	return err
}

// Create a Rest client with the new CRD Schema
var SchemeGroupVersion = schema.GroupVersion{Group: LBGroup, Version: LBVersion}

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&AppLoadBalance{},
		&AppLoadBalanceList{},
	)
	meta_v1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}