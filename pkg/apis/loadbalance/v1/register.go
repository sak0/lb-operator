package v1

import (
	"reflect"

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
	ALBPlural      string = "apploadbalance"
	ALBGroup       string = "loadbalance.yonghui.cn"
	ALBVersion     string = "v1"
	FullALBName    string = ALBPlural + "." + ALBGroup
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
			Group:   ALBGroup,
			Version: ALBVersion,
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
		return nil
	}
	return err
}

// Create a Rest client with the new CRD Schema
var SchemeGroupVersion = schema.GroupVersion{Group: ALBGroup, Version: ALBVersion}

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&AppLoadBalance{},
		&AppLoadBalanceList{},
	)
	meta_v1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}