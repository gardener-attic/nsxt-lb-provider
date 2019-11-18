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
	"github.com/pkg/errors"
	"github.com/vmware/go-vmware-nsxt/loadbalancer"

	"github.com/gardener/nsxt-lb-provider/pkg/nsxt-lb-provider/config"
	nsxt "github.com/vmware/go-vmware-nsxt"
	corev1 "k8s.io/api/core/v1"
	cloudprovider "k8s.io/cloud-provider"
)

type lbProvider struct {
	access  Access
	keyLock *KeyLock
}

// TODO cluster name needed for reorg and is currently injected from main
var ClusterName string

var _ cloudprovider.LoadBalancer = &lbProvider{}

func newLBProvider(config *config.Config) (*lbProvider, error) {
	nsxtConfig := config.NSXT
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

	nsxClient, err := nsxt.NewAPIClient(&cfg)
	if err != nil {
		return nil, errors.Wrap(err, "creating NSX-T client failed")
	}

	access, err := NewAccess(nsxClient, config)
	if err != nil {
		return nil, errors.Wrap(err, "creating access handler failed")
	}
	return &lbProvider{access: access, keyLock: NewKeyLock()}, nil
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

	state, err := newState(clusterName, service, p.access)
	if err != nil {
		return nil, err
	}
	for _, servicePort := range service.Spec.Ports {
		mapping := NewMapping(servicePort)
		found := false
		for _, server := range state.servers {
			if mapping.MatchVirtualServer(server) {
				err = p.updateVirtualServer(server, mapping, nodes, state)
				if err != nil {
					return nil, err
				}
				found = true
				break
			}
		}
		if !found {
			err = p.createVirtualServer(mapping, nodes, state)
			if err != nil {
				return nil, err
			}
		}
	}
	for _, server := range state.servers {
		found := false
		for _, servicePort := range service.Spec.Ports {
			mapping := NewMapping(servicePort)
			if mapping.MatchVirtualServer(server) {
				found = true
				break
			}
		}
		if !found {
			err = p.deleteVirtualServer(server, state)
			if err != nil {
				return nil, err
			}
		}
	}
	return state.finish()
}

func (p *lbProvider) createVirtualServer(mapping Mapping, nodes []*corev1.Node, state *state) error {
	lbService, err := p.access.FindFreeLoadBalancerService(state.clusterName)
	if err != nil {
		return err
	}
	if lbService == nil {
		lbService, err = p.access.CreateLoadBalancerService(state.clusterName)
		if err != nil {
			return err
		}
	}
	err = state.initialize()
	if err != nil {
		return err
	}
	vserver, err := p.access.CreateVirtualServer(state.clusterName, objectNameFromService(state.service), state.ipAddress, mapping, state.poolID)
	if err != nil {
		_, _ = state.finish()
		return err
	}
	lbService.VirtualServerIds = append(lbService.VirtualServerIds, vserver.Id)
	err = p.access.UpdateLoadBalancerService(lbService)
	if err != nil {
		return err
	}
	return nil
}

func collectNodeInternalAddresses(nodes []*corev1.Node) map[string]string {
	set := map[string]string{}
	for _, node := range nodes {
		for _, addr := range node.Status.Addresses {
			if addr.Type == corev1.NodeInternalIP {
				set[addr.Address] = node.Name
				break
			}
		}
	}
	return set
}

func (p *lbProvider) updateVirtualServer(server *loadbalancer.LbVirtualServer, mapping Mapping, nodes []*corev1.Node, state *state) error {
	if !mapping.MatchNodePort(server) {
		server.DefaultPoolMemberPort = formatPort(mapping.NodePort)
		err := p.access.UpdateVirtualServer(server)
		if err != nil {
			return err
		}
	}
	pool, err := state.getPool()
	if err != nil {
		return err
	}
	return p.updatePoolMembers(state.clusterName, pool, nodes)
}

func (p *lbProvider) updatePoolMembers(clusterName string, pool *loadbalancer.LbPool, nodes []*corev1.Node) error {
	modified := false
	nodeIpAddresses := collectNodeInternalAddresses(nodes)
	newMembers := []loadbalancer.PoolMember{}
	for _, member := range pool.Members {
		if _, ok := nodeIpAddresses[member.IpAddress]; ok {
			newMembers = append(newMembers, member)
		} else {
			modified = true
		}
	}
	if len(nodeIpAddresses) > len(newMembers) {
		for nodeIpAddress, nodeName := range nodeIpAddresses {
			found := false
			for _, member := range pool.Members {
				if member.IpAddress == nodeIpAddress {
					found = true
					break
				}
			}
			if !found {
				member := loadbalancer.PoolMember{
					AdminState:  "ENABLED",
					DisplayName: fmt.Sprintf("%s:%s", clusterName, nodeName),
					IpAddress:   nodeIpAddress,
				}
				newMembers = append(newMembers, member)
				modified = true
			}
		}
	}
	if modified {
		pool.Members = newMembers
		err := p.access.UpdatePool(pool)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *lbProvider) deleteVirtualServer(server *loadbalancer.LbVirtualServer, state *state) error {
	lbService, err := p.access.FindLoadBalancerServiceForVirtualServer(state.clusterName, server.Id)
	if err != nil {
		return err
	}
	if lbService != nil {
		for i, id := range lbService.VirtualServerIds {
			if id == server.Id {
				lbService.VirtualServerIds = append(lbService.VirtualServerIds[:i], lbService.VirtualServerIds[i+1:]...)
				break
			}
		}
		if len(lbService.VirtualServerIds) == 0 {
			err := p.access.DeleteLoadBalancerService(lbService.Id)
			if err != nil {
				return err
			}
		} else {
			err := p.access.UpdateLoadBalancerService(lbService)
			if err != nil {
				return err
			}
		}
	}
	return p.access.DeleteVirtualServer(server.Id)
}

// UpdateLoadBalancer updates hosts under the specified load balancer.
// Implementations must treat the *corev1.Service and *corev1.Node
// parameters as read-only and not modify them.
// Parameter 'clusterName' is the name of the cluster as presented to kube-controller-manager
func (p *lbProvider) UpdateLoadBalancer(ctx context.Context, clusterName string, service *corev1.Service, nodes []*corev1.Node) error {
	key := objectNameFromService(service).String()
	p.keyLock.Lock(key)
	defer p.keyLock.Unlock(key)

	pool, err := p.access.FindPool(clusterName, objectNameFromService(service))
	if err != nil {
		return err
	}
	return p.updatePoolMembers(clusterName, pool, nodes)
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
