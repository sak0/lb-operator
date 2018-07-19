package drivers

import (
	"fmt"
	"strings"	
	
	"github.com/golang/glog"
	
	/*"github.com/chiradeep/go-nitro/config/basic"
	"github.com/chiradeep/go-nitro/config/cs"
	"github.com/chiradeep/go-nitro/config/lb"
	"github.com/chiradeep/go-nitro/netscaler"*/
)

const (
	CITRIXLBPROVIDER	 = "citrix"
)

type LbProvider interface{
	CreateLb(string, string, string)(string, error)
	DeleteLb(string)error
	CreateSvcGroup(string)error
	BindSvcGroupLb(string, string)error
}

func GenerateLbName(namespace string, host string) string {
	lbName := "lb_" + strings.Replace(host, ".", "_", -1)
	return lbName
}

func GenerateLbNameNew(namespace string, host string, path string) string {
	path_ := path
	if path == "" {
		path_ = "nilpath"
	}
	path_ = strings.Replace(path_, "/", "_", -1)
		
	lbName := "lb_" + strings.Replace(host, ".", "_", -1) + path_
	return lbName
}

func GenerateCsVserverName(namespace string, ingressName string) string {
	csv := "cs_" + namespace + "_" + ingressName
	return csv
}

type CitrixLb struct{}
func (lb *CitrixLb)CreateLb(vip string, port string, protocol string)(string, error) {
	glog.V(2).Infof("Citrix Driver CreateLB..")
	return "testname", nil
}
func (lb *CitrixLb)CreateSvcGroup(groupname string)error{
	return nil
}
func (lb *CitrixLb)BindSvcGroupLb(groupname string, lbname string)error{
	return nil
}
func (lb *CitrixLb)DeleteLb(lbname string)error{
	glog.V(2).Infof("Citrix Driver DeleteLb..")
	return nil
}

func New(lbtype string)(LbProvider, error){
	switch lbtype {
		case CITRIXLBPROVIDER:
			return &CitrixLb{}, nil
		default:
			return nil, fmt.Errorf("Unsupport type: %s", lbtype)
	}
}