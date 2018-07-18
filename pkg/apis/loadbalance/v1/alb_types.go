package v1

import (
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Definition of our CRD AppLoadBalance class
type AppLoadBalance struct {
	meta_v1.TypeMeta   `json:",inline"`
	meta_v1.ObjectMeta `json:"metadata"`
	Spec               AppLoadBalanceSpec   `json:"spec"`
	Status             AppLoadBalanceStatus `json:"status,omitempty"`
}

type AppLoadBalanceSpec struct {
	IP		string					`json:"ip,omitempty"`
	Port	string					`json:"port,omitempty"`
	Rules	[]AppLoadBalanceRule	`json:"rules,omitempty"`
}

type AppLoadBalanceRule struct {
	Host	string					`json:"host,omitempty"`
	Paths	[]AppLoadBalancePath	`json:"paths,omitempty"`
}

type AppLoadBalancePath struct {
	Path	string					`json:"path,omitempty"`
	Backend	AppLoadBalanceBackend	`json:"backend"`
}

type AppLoadBalanceBackend struct {
	ServiceName	string	`json:"serviceName"`
	ServicePort	int		`json:"servicePort"`
}

type AppLoadBalanceStatus struct {
	State   string `json:"state,omitempty"`
	Message string `json:"message,omitempty"`
}

type AppLoadBalanceList struct {
	meta_v1.TypeMeta `json:",inline"`
	meta_v1.ListMeta `json:"metadata"`
	Items            []AppLoadBalance `json:"items"`
}