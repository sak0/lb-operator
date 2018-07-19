package v1

import (
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Definition of our CRD AppLoadBalance class
type ClassicLoadBalance struct {
	meta_v1.TypeMeta   `json:",inline"`
	meta_v1.ObjectMeta `json:"metadata"`
	Spec               ClassicLoadBalanceSpec   `json:"spec"`
	Status             ClassicLoadBalanceStatus `json:"status,omitempty"`
}

type ClassicLoadBalanceSpec struct {
	IP			string						`json:"ip,omitempty"`
	Port		string						`json:"port"`
	Subnet		string						`json:"subnet,omitempty"`
	Protocol	string						`json:"protocol"`
	Backends	[]ClassicLoadBalanceBackend	`json:"backends,omitempty"`
}

type ClassicLoadBalanceBackend struct {
	ServiceName	string	`json:"serviceName"`
	ServicePort	int		`json:"servicePort"`
}

type ClassicLoadBalanceStatus struct {
	State   string `json:"state,omitempty"`
	Message string `json:"message,omitempty"`
}

type ClassicLoadBalanceList struct {
	meta_v1.TypeMeta `json:",inline"`
	meta_v1.ListMeta `json:"metadata"`
	Items            []ClassicLoadBalance `json:"items"`
}