package drivers

import (
	"fmt"
	"strconv"
	"strings"	
	
	"github.com/golang/glog"
	
	"github.com/sak0/lb-operator/pkg/utils"
	
	citrixbasic "github.com/chiradeep/go-nitro/config/basic"
	/*"github.com/chiradeep/go-nitro/config/cs"*/
	citrixlb "github.com/chiradeep/go-nitro/config/lb"
	"github.com/chiradeep/go-nitro/netscaler"
)

const (
	CITRIXLBPROVIDER	 = "citrix"
)

type LbProvider interface {
	CreateLb(string, string, string, string)(string, error)
	DeleteLb(string, string)error
	CreateSvcGroup(string, string)error
	BindSvcGroupLb(string, string, string)error
	CreateSvc(string, string, int32, string)(string, error)
	BindSvcToLb(string, string)error
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
func (lb *CitrixLb)CreateLb(namespace string, vip string, port string, protocol string)(string, error) {
	lbName := utils.GenerateLbNameCLB(namespace, vip, port, protocol)
	portint, err := strconv.Atoi(port)
	if err != nil {
		return "", fmt.Errorf("Convert port %v to int failed: %v", port, err)
	}
	
	glog.V(2).Infof("Citrix Driver CreateLB..")
	
	glog.V(2).Infof("CreateLB Lbvserver %s", lbName)
	client, _ := netscaler.NewNitroClientFromEnv()
	nsLB := citrixlb.Lbvserver{
		Name			: lbName,
		Ipv46			: vip,
		Port			: portint,
		Servicetype		: protocol,
	}
	_, _ = client.AddResource(netscaler.Lbvserver.Type(), lbName, &nsLB)	
	
	return lbName, nil
}
func (lb *CitrixLb)CreateSvcGroup(namespace string, groupname string)error{
	return nil
}
func (lb *CitrixLb)BindSvcGroupLb(namespace string, groupname string, lbname string)error{
	return nil
}
func (lb *CitrixLb)DeleteLb(namespace string, lbname string)error{
	glog.V(2).Infof("Citrix Driver DeleteLb..")
	return nil
}
func (lb *CitrixLb)CreateSvc(namespace string, ip string, port int32, protocol string)(string, error){
	svcName := utils.GenerateSvcNameCLB(namespace, ip, port, protocol)
	glog.V(2).Infof("Citrix Driver CreateSvc %s ", svcName)
	
	client, _ := netscaler.NewNitroClientFromEnv()
	nsService := citrixbasic.Service{
		Name:        svcName,
		Ip:          ip,
		Servicetype: protocol,
		Port:        int(port),
	}
	_, err := client.AddResource(netscaler.Service.Type(), svcName, &nsService)
	if err != nil {
		return "", err
	}
	
	return svcName, nil
}

func (lb *CitrixLb)BindSvcToLb(svcName string, lbName string)error{
	glog.V(2).Infof("Citrix Driver Bind Svc %s to %s", svcName, lbName)
	
	client, _ := netscaler.NewNitroClientFromEnv()
	binding := citrixlb.Lbvserverservicebinding{
		Name:        lbName,
		Servicename: svcName,
	}
	err := client.BindResource(netscaler.Lbvserver.Type(), lbName, netscaler.Service.Type(), svcName, &binding)
	if err != nil {
		return err
	} 
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