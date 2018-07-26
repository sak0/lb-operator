package utils

import (
	"os"
	"fmt"
	
	"github.com/golang/glog"
	
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	//"github.com/gophercloud/gophercloud/openstack/utils"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	//"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"	
)


const (
	DEFAULTDOMAIN 	= "Default"
	DEFAULTREGION 	= "RegionOne"
	DEFAULTTENANT 	= "admin"
	DEFAULTUSER		= "admin"
)

type OpsClient struct{
	Endpoint		string
	Username		string
	Password		string
	Domain			string
	TenantID		string
	
	Client			*gophercloud.ServiceClient
}

func NewOpsClient()(*OpsClient, error){
	pass := os.Getenv("OS_PASSWORD")
	ep := os.Getenv("OS_AUTH_URL")
	tenantId := os.Getenv("OS_TENANT_ID")
	
	opts := gophercloud.AuthOptions {
		IdentityEndpoint: ep,
		Username: DEFAULTUSER,
		Password: pass,
		DomainName: DEFAULTDOMAIN,
		TenantID: tenantId,
	}
	provider, err := openstack.AuthenticatedClient(opts)
	if err != nil {
		return nil, err
	}
	client, err := openstack.NewNetworkV2(provider, gophercloud.EndpointOpts{
		Region: DEFAULTREGION,
	})
	if err != nil {
		return nil, err
	}
	oc := &OpsClient {
		Endpoint : ep,
		Username : DEFAULTUSER,
		Password : pass,
		Domain: DEFAULTDOMAIN,
		TenantID: tenantId,
		Client	: client,
	}
	return oc, nil
}

func (c *OpsClient)GetNetId(subnetId string)(string, error){
	rs := subnets.Get(c.Client, subnetId)
	subnet, err := rs.Extract()
	if err != nil {
		return "", err
	}
	return subnet.NetworkID, nil
}

func (c *OpsClient)GetIpAddr(netid string)(ipaddr string, mac string, err error){
	pcOpts := ports.CreateOpts{
		//Name	  : cid,
		NetworkID : netid,
	}
	res := ports.Create(c.Client, pcOpts)
	port, err := res.Extract()
	if err != nil{
		return "", "", nil
	}
	return port.FixedIPs[0].IPAddress, port.ID, nil
}

func (c *OpsClient)UpdatePortName(portId string, portName string)error {
	puOpts := ports.UpdateOpts{
		Name : portName,
	}
	res := ports.Update(c.Client, portId, puOpts)
	_, err := res.Extract()
	if err != nil{
		return err
	}
	return nil
}

func (c *OpsClient)DeletePort(portName string)error {
	lsOpts := ports.ListOpts{
		Name : portName,
	}
	allPages, err := ports.List(c.Client, lsOpts).AllPages()
	if err != nil {
		return err
	}
	allPorts, err := ports.ExtractPorts(allPages)
	if err != nil {
		return err
	}
	if len(allPorts) > 1 {
		glog.Errorf("Port named %s > 1.", portName)
	} else if len(allPorts) < 1{
		return nil
	}
	pid := allPorts[0].ID
	dres := ports.Delete(c.Client, pid)
	return dres.Err	
}

func (c *OpsClient)CreatePort(ip string, netid string, subnetId string)(string, string, error) {
	portIP := ports.IP{
		SubnetID : subnetId,
		IPAddress : ip,
	}
	pcOpts := ports.CreateOpts{
		//Name	  : cid,
		NetworkID : netid,
		FixedIPs  : []ports.IP{
			portIP,
		},
	}
	res := ports.Create(c.Client, pcOpts)
	port, err := res.Extract()
	if err != nil{
		return "", "", err
	}
	return port.FixedIPs[0].IPAddress, port.ID, nil
}	

func AllocIpAddrFromSubnet(namespace string, subnetId string)(string, error){
	opClient, err := NewOpsClient()
	if err != nil {
		return "", fmt.Errorf("Get Openstack client failed: %v", err)
	}
	
	netId, err := opClient.GetNetId(subnetId)
	if err != nil {
		return "", fmt.Errorf("Get network id failed: %v", err)
	}
	
	ip, portid, err := opClient.GetIpAddr(netId)
	if err != nil {
		return "", fmt.Errorf("Create port failed: %v", err)
	}
	portName := GeneratePortNameCLB(namespace, ip)
	err = opClient.UpdatePortName(portid, portName)
	if err != nil {
		glog.Errorf("Update port failed: %v", err)
	}
	
	return ip, nil
}

func ReleaseIpAddr(namespace string, vip string)error {
	opClient, err := NewOpsClient()
	if err != nil {
		return fmt.Errorf("Get Openstack client failed: %v", err)
	}
	
	portName := GeneratePortNameCLB(namespace, vip)	
	err = opClient.DeletePort(portName)
	if err != nil {
		glog.Errorf("Delete port failed: %v", err)	
	}
	
	return nil
}

func CreatePortFromIp(namespace string, ip string, subnetId string)error {
	opClient, err := NewOpsClient()
	if err != nil {
		return fmt.Errorf("Get Openstack client failed: %v", err)
	}

	netId, err := opClient.GetNetId(subnetId)
	if err != nil {
		return fmt.Errorf("Get network id failed: %v", err)
	}
	
	ip, portid, err := opClient.CreatePort(ip, netId, subnetId)
	if err != nil {
		return fmt.Errorf("Create port failed: %v", err)
	}
	portName := GeneratePortNameCLB(namespace, ip)
	err = opClient.UpdatePortName(portid, portName)
	if err != nil {
		glog.Errorf("Update port failed: %v", err)
	}	
	
	return nil			
}