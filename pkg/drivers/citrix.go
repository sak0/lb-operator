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
	DeleteLb(string, string, string, string)error
	CreateSvcGroup(string, string)(string, error)
	BindSvcGroupLb(string, string)error
	CreateSvc(string, string, int32, string)(string, error)
	BindSvcToLb(string, string, int)error
	CreateServer(string, string)(string, error)
	BindServerToGroup(string, string, int32, int)error
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
	
	glog.V(2).Infof("Citrix create Lbvserver %s", lbName)
	client, _ := netscaler.NewNitroClientFromEnv()
	nsLB := citrixlb.Lbvserver{
		Name			: lbName,
		Ipv46			: vip,
		Port			: portint,
		Servicetype		: protocol,
		Lbmethod        : "ROUNDROBIN",
	}
	_, _ = client.AddResource(netscaler.Lbvserver.Type(), lbName, &nsLB)	
	
	return lbName, nil
}

func (lb *CitrixLb)CreateSvcGroup(namespace string, svcname string)(string, error){
	groupName := utils.GenerateSvcGroupNameCLB(namespace, svcname)
	
	glog.V(2).Infof("Citrix Driver CreateSvcGroup")
	client, _ := netscaler.NewNitroClientFromEnv()
	nsSvcGrp := citrixbasic.Servicegroup{
		Servicegroupname	: groupName,
		//TODO: resolve k8s svc only support tcp udp protocol
		Servicetype			: "TCP",
	}
	_, err := client.AddResource(netscaler.Servicegroup.Type(), groupName, &nsSvcGrp)
	if err != nil {
		return groupName, err
	}
	return groupName, nil
}

func (lb *CitrixLb)BindSvcGroupLb(groupname string, lbname string)error{
	glog.V(2).Infof("Citrix Driver BindSvcGroupLb. bind %s to %s", groupname, lbname)
	client, _ := netscaler.NewNitroClientFromEnv()
	binding := citrixlb.Lbvserverservicegroupbinding{
		Servicegroupname	: groupname,
		Name				: lbname,
		//Weight				: weight,
	}
	err := client.BindResource(netscaler.Lbvserver.Type(), lbname, netscaler.Servicegroup.Type(), groupname, &binding)
	if err != nil {
		return err
	} 
	return nil
}

func (lb *CitrixLb)DeleteLb(namespace string, vip string, port string, protocol string)error{
	glog.V(2).Infof("Citrix Driver DeleteLb..")
	lbName := utils.GenerateLbNameCLB(namespace, vip, port, protocol)
	
	client, _ := netscaler.NewNitroClientFromEnv()
	err := client.DeleteResource(netscaler.Lbvserver.Type(), lbName)
	if err != nil {
		return err
	}
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
		return svcName, err
	}
	
	return svcName, nil
}

func (lb *CitrixLb)BindSvcToLb(svcName string, lbName string, weight int)error{
	glog.V(2).Infof("Citrix Driver Bind Svc %s to %s", svcName, lbName)
	
	client, _ := netscaler.NewNitroClientFromEnv()
	binding := citrixlb.Lbvserverservicebinding{
		Name:        lbName,
		Servicename: svcName,
		Weight:		 weight,
	}
	err := client.BindResource(netscaler.Lbvserver.Type(), lbName, netscaler.Service.Type(), svcName, &binding)
	if err != nil {
		return err
	} 
	return nil
}

func (lb *CitrixLb)CreateServer(namespace string, ip string)(string, error){
	glog.V(2).Infof("Citrix Driver CreateServer %s", ip)
	serverName := utils.GenerateServerNameCLB(namespace, ip)
	
	client, _ := netscaler.NewNitroClientFromEnv()
	nsServer := citrixbasic.Server{
		Name			: serverName,
		Ipaddress		: ip,
	}
	_, err := client.AddResource(netscaler.Server.Type(), serverName, &nsServer)
	if err != nil {
		return serverName, err
	}	
	return serverName, nil
}

func (lb *CitrixLb)BindServerToGroup(serverName string, groupName string, port int32, weight int)error{
	glog.V(2).Infof("*********Citrix Driver BindServerToGroup %s->%s", serverName, groupName)

	client, _ := netscaler.NewNitroClientFromEnv()
	binding := citrixbasic.Servicegroupservicegroupmemberbinding{
		Servicegroupname	: groupName,
		Servername			: serverName,
		Port				: int(port),
		Weight				: weight,
	}
	//err := client.BindResource(netscaler.Servicegroup.Type(), groupName, netscaler.Server.Type(), serverName, &binding)
	_, err := client.AddResource(netscaler.Servicegroup_servicegroupmember_binding.Type(), groupName, &binding)
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