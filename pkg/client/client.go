package client

import (
	crdv1 "github.com/sak0/lb-operator/pkg/apis/loadbalance/v1"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

func AlbClient(cl *rest.RESTClient, scheme *runtime.Scheme, namespace string) *albclient {
	return &albclient{cl: cl, ns: namespace, plural: crdv1.ALBPlural,
		codec: runtime.NewParameterCodec(scheme)}
}

type albclient struct {
	cl		*rest.RESTClient
	ns		string
	plural	string
	codec	runtime.ParameterCodec
}

func (f *albclient) Create(obj *crdv1.AppLoadBalance) (*crdv1.AppLoadBalance, error) {
	var result crdv1.AppLoadBalance
	err := f.cl.Post().
		Namespace(f.ns).Resource(f.plural).
		Body(obj).Do().Into(&result)
	return &result, err
}

func (f *albclient) Update(obj *crdv1.AppLoadBalance) (*crdv1.AppLoadBalance, error) {
	var result crdv1.AppLoadBalance
	err := f.cl.Put().
		Namespace(f.ns).Resource(f.plural).
		Body(obj).Do().Into(&result)
	return &result, err
}

func (f *albclient) Delete(name string, options *meta_v1.DeleteOptions) error {
	return f.cl.Delete().
		Namespace(f.ns).Resource(f.plural).
		Name(name).Body(options).Do().
		Error()
}

func (f *albclient) Get(name string) (*crdv1.AppLoadBalance, error) {
	var result crdv1.AppLoadBalance
	err := f.cl.Get().
		Namespace(f.ns).Resource(f.plural).
		Name(name).Do().Into(&result)
	return &result, err
}

func (f *albclient) List(opts meta_v1.ListOptions) (*crdv1.AppLoadBalanceList, error) {
	var result crdv1.AppLoadBalanceList
	err := f.cl.Get().
		Namespace(f.ns).Resource(f.plural).
		VersionedParams(&opts, f.codec).
		Do().Into(&result)
	return &result, err
}

// Create a new List watch for our TPR
func (f *albclient) NewListWatch() *cache.ListWatch {
	//return cache.NewListWatchFromClient(f.cl, f.plural, f.ns, fields.Everything())
	return cache.NewListWatchFromClient(f.cl, f.plural, meta_v1.NamespaceAll, fields.Everything())
}

func ClbClient(cl *rest.RESTClient, scheme *runtime.Scheme, namespace string) *clbclient {
	return &clbclient{cl: cl, ns: namespace, plural: crdv1.CLBPlural, 
		codec: runtime.NewParameterCodec(scheme)}
}

type clbclient struct {
	cl		*rest.RESTClient
	ns		string
	plural	string
	codec 	runtime.ParameterCodec
}

func (f *clbclient) Create(obj *crdv1.ClassicLoadBalance) (*crdv1.ClassicLoadBalance, error) {
	var result crdv1.ClassicLoadBalance
	err := f.cl.Post().
		Namespace(f.ns).Resource(f.plural).
		Body(obj).Do().Into(&result)
	return &result, err
}

func (f *clbclient) Update(obj *crdv1.ClassicLoadBalance, name string) (*crdv1.ClassicLoadBalance, error) {
	var result crdv1.ClassicLoadBalance
	err := f.cl.Put().
		Namespace(f.ns).Resource(f.plural).
		Name(name).
		Body(obj).Do().Into(&result)
	return &result, err
}

func (f *clbclient) Delete(name string, options *meta_v1.DeleteOptions) error {
	return f.cl.Delete().
		Namespace(f.ns).Resource(f.plural).
		Name(name).Body(options).Do().
		Error()
}

func (f *clbclient) Get(name string) (*crdv1.ClassicLoadBalance, error) {
	var result crdv1.ClassicLoadBalance
	err := f.cl.Get().
		Namespace(f.ns).Resource(f.plural).
		Name(name).Do().Into(&result)
	return &result, err
}

func (f *clbclient) List(opts meta_v1.ListOptions) (*crdv1.ClassicLoadBalanceList, error) {
	var result crdv1.ClassicLoadBalanceList
	err := f.cl.Get().
		Namespace(f.ns).Resource(f.plural).
		VersionedParams(&opts, f.codec).
		Do().Into(&result)
	return &result, err
}

// Create a new List watch for our TPR
func (f *clbclient) NewListWatch() *cache.ListWatch {
	//return cache.NewListWatchFromClient(f.cl, f.plural, f.ns, fields.Everything())
	return cache.NewListWatchFromClient(f.cl, f.plural, meta_v1.NamespaceAll, fields.Everything())
}


func NewClient(cfg *rest.Config) (*rest.RESTClient, *runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	if err := crdv1.AddToScheme(scheme); err != nil {
		return nil, nil, err
	}
	
	config := *cfg
	config.GroupVersion = &crdv1.SchemeGroupVersion
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.DirectCodecFactory{
		CodecFactory: serializer.NewCodecFactory(scheme)}

	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, nil, err
	}
	return client, scheme, nil
}
