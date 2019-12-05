/*
 * Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package nsxt_lb_provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/gardener/nsxt-lb-provider/pkg/nsxt-lb-provider/config"
	"github.com/pkg/errors"
	nsxt "github.com/vmware/go-vmware-nsxt"
	corev1 "k8s.io/api/core/v1"
	cloudprovider "k8s.io/cloud-provider"
)

const (
	AnnotLoadBalancerClass = "loadbalancer.vsphere.class"
)

type lbProvider struct {
	*lbService
	classes *loadBalancerClasses
	keyLock *KeyLock
}

// TODO cluster name needed for reorg and is currently injected from main
var ClusterName string

var _ cloudprovider.LoadBalancer = &lbProvider{}

func newLBProvider(config *config.Config) (*lbProvider, error) {
	broker, err := setupNsxtBroker(&config.NSXT)
	if err != nil {
		return nil, err
	}
	access, err := NewAccess(broker, config)
	if err != nil {
		return nil, errors.Wrap(err, "creating access handler failed")
	}
	classes, err := setupClasses(access, config)
	if err != nil {
		return nil, errors.Wrap(err, "creating load balancer classes failed")
	}
	return &lbProvider{lbService: newLbService(access), classes: classes, keyLock: NewKeyLock()}, nil
}

func setupNsxtBroker(nsxtConfig *config.NsxtConfig) (NsxtBroker, error) {
	if nsxtConfig.SimulateInMemory {
		return NewInMemoryNsxtBroker(nsxtConfig.SimulatedIPPools...), nil
	} else {
		retriesConfig := nsxt.ClientRetriesConfiguration{
			MaxRetries:      nsxtConfig.MaxRetries,
			RetryMinDelay:   nsxtConfig.RetryMinDelay,
			RetryMaxDelay:   nsxtConfig.RetryMaxDelay,
			RetryOnStatuses: nsxtConfig.RetryOnStatusCodes,
		}
		cfg := nsxt.Configuration{
			BasePath:             "/api/v1",
			Host:                 nsxtConfig.Host,
			Scheme:               "https",
			UserAgent:            "nsxt-lb-provider/" + Version,
			UserName:             nsxtConfig.User,
			Password:             nsxtConfig.Password,
			RemoteAuth:           nsxtConfig.RemoteAuth,
			ClientAuthCertFile:   nsxtConfig.ClientAuthCertFile,
			ClientAuthKeyFile:    nsxtConfig.ClientAuthKeyFile,
			CAFile:               nsxtConfig.CAFile,
			Insecure:             nsxtConfig.InsecureFlag,
			RetriesConfiguration: retriesConfig,
		}
		client, err := nsxt.NewAPIClient(&cfg)
		if err != nil {
			return nil, errors.Wrap(err, "creating NSX-T client failed")
		}
		return NewNsxtBroker(client), nil
	}
}

func (p *lbProvider) Initialize(clientBuilder cloudprovider.ControllerClientBuilder, stop <-chan struct{}) {
	client, err := clientBuilder.Client("reorg")
	if err != nil {
		panic(err)
	}
	go p.reorg(client.CoreV1().Services(""), stop)
}

// Implementations must treat the *corev1.Service parameter as read-only and not modify it.
// Parameter 'clusterName' is the name of the cluster as presented to kube-controller-manager
func (p *lbProvider) GetLoadBalancer(ctx context.Context, clusterName string, service *corev1.Service) (status *corev1.LoadBalancerStatus, exists bool, err error) {
	servers, err := p.access.FindVirtualServers(clusterName, objectNameFromService(service))
	if err != nil {
		return nil, false, err
	}
	if len(servers) == 0 {
		return nil, false, nil
	}
	return newLoadBalancerStatus(servers[0].IpAddress), true, nil
}

func newLoadBalancerStatus(ipAddress string) *corev1.LoadBalancerStatus {
	return &corev1.LoadBalancerStatus{
		Ingress: []corev1.LoadBalancerIngress{
			{IP: ipAddress},
		},
	}
}

// GetLoadBalancerName returns the name of the load balancer. Implementations must treat the
// *corev1.Service parameter as read-only and not modify it.
func (p *lbProvider) GetLoadBalancerName(ctx context.Context, clusterName string, service *corev1.Service) string {
	return clusterName + ":" + service.Namespace + ":" + service.Name
}

// EnsureLoadBalancer creates a new load balancer 'name', or updates the existing one. Returns the status of the balancer
// Implementations must treat the *corev1.Service and *corev1.Node
// parameters as read-only and not modify them.
// Parameter 'clusterName' is the name of the cluster as presented to kube-controller-manager
func (p *lbProvider) EnsureLoadBalancer(ctx context.Context, clusterName string, service *corev1.Service, nodes []*corev1.Node) (*corev1.LoadBalancerStatus, error) {
	key := objectNameFromService(service).String()
	p.keyLock.Lock(key)
	defer p.keyLock.Unlock(key)

	class, err := p.classFromService(service)
	if err != nil {
		return nil, err
	}

	state := NewState(p.lbService, clusterName, service, nodes)
	err = state.Process(class)
	status, err2 := state.Finish()
	if err != nil {
		return status, err
	}
	return status, err2
}

func (p *lbProvider) classFromService(service *corev1.Service) (*loadBalancerClass, error) {
	annos := service.GetAnnotations()
	if annos == nil {
		annos = map[string]string{}
	}
	name, ok := annos[AnnotLoadBalancerClass]
	name = strings.TrimSpace(name)
	if !ok || name == "" {
		name = config.DefaultLoadBalancerClass
	}

	class := p.classes.GetClass(name)
	if class == nil {
		return nil, fmt.Errorf("invalid load balancer class %s", name)
	}
	return class, nil
}

// UpdateLoadBalancer updates hosts under the specified load balancer.
// Implementations must treat the *corev1.Service and *corev1.Node
// parameters as read-only and not modify them.
// Parameter 'clusterName' is the name of the cluster as presented to kube-controller-manager
func (p *lbProvider) UpdateLoadBalancer(ctx context.Context, clusterName string, service *corev1.Service, nodes []*corev1.Node) error {
	key := objectNameFromService(service).String()
	p.keyLock.Lock(key)
	defer p.keyLock.Unlock(key)

	state := NewState(p.lbService, clusterName, service, nodes)

	return state.UpdatePoolMembers()
}

// EnsureLoadBalancerDeleted deletes the specified load balancer if it
// exists, returning nil if the load balancer specified either didn't exist or
// was successfully deleted.
// This construction is useful because many cloud providers' load balancers
// have multiple underlying components, meaning a Get could say that the LB
// doesn't exist even if some part of it is still laying around.
// Implementations must treat the *corev1.Service parameter as read-only and not modify it.
// Parameter 'clusterName' is the name of the cluster as presented to kube-controller-manager
func (p *lbProvider) EnsureLoadBalancerDeleted(ctx context.Context, clusterName string, service *corev1.Service) error {
	emptyService := *service
	emptyService.Spec.Ports = nil
	_, err := p.EnsureLoadBalancer(ctx, clusterName, &emptyService, nil)
	return err
}
