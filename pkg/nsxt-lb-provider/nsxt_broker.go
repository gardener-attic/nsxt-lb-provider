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
	nsxt "github.com/vmware/go-vmware-nsxt"
	"github.com/vmware/go-vmware-nsxt/loadbalancer"
	"github.com/vmware/go-vmware-nsxt/manager"
)

type NsxtBroker interface {
	ListIpPools() (manager.IpPoolListResult, error)
	CreateLoadBalancerService(service loadbalancer.LbService) (loadbalancer.LbService, error)
	ListLoadBalancerServices() (loadbalancer.LbServiceListResult, error)
	UpdateLoadBalancerService(service loadbalancer.LbService) (loadbalancer.LbService, error)
	DeleteLoadBalancerService(id string) (statusCode int, err error)
	CreateLoadBalancerVirtualServer(server loadbalancer.LbVirtualServer) (loadbalancer.LbVirtualServer, error)
	ListLoadBalancerVirtualServers() (loadbalancer.LbVirtualServerListResult, error)
	UpdateLoadBalancerVirtualServer(server loadbalancer.LbVirtualServer) (loadbalancer.LbVirtualServer, error)
	DeleteLoadBalancerVirtualServer(id string) (statusCode int, err error)
	CreateLoadBalancerPool(pool loadbalancer.LbPool) (loadbalancer.LbPool, error)
	ReadLoadBalancerPool(id string) (loadbalancer.LbPool, error)
	ListLoadBalancerPools() (loadbalancer.LbPoolListResult, error)
	UpdateLoadBalancerPool(pool loadbalancer.LbPool) (loadbalancer.LbPool, error)
	DeleteLoadBalancerPool(id string) (statusCode int, err error)
	AllocateFromIpPool(ipPoolID string) (ipAddress string, statusCode int, err error)
	ListIpPoolAllocations(ipPoolID string) (ipAddresses []string, statusCode int, err error)
	ReleaseFromIpPool(ipPoolID, ipAddress string) (statusCode int, err error)
}

type nsxtBroker struct {
	client *nsxt.APIClient
}

func NewNsxtBroker(client *nsxt.APIClient) NsxtBroker {
	return &nsxtBroker{client: client}
}

func (b *nsxtBroker) ListIpPools() (manager.IpPoolListResult, error) {
	result, _, err := b.client.PoolManagementApi.ListIpPools(b.client.Context, nil)
	return result, err
}

func (b *nsxtBroker) CreateLoadBalancerService(service loadbalancer.LbService) (loadbalancer.LbService, error) {
	result, _, err := b.client.ServicesApi.CreateLoadBalancerService(b.client.Context, service)
	return result, err
}

func (b *nsxtBroker) ListLoadBalancerServices() (loadbalancer.LbServiceListResult, error) {
	result, _, err := b.client.ServicesApi.ListLoadBalancerServices(b.client.Context, nil)
	return result, err
}

func (b *nsxtBroker) UpdateLoadBalancerService(service loadbalancer.LbService) (loadbalancer.LbService, error) {
	result, _, err := b.client.ServicesApi.UpdateLoadBalancerService(b.client.Context, service.Id, service)
	return result, err
}

func (b *nsxtBroker) DeleteLoadBalancerService(id string) (int, error) {
	resp, err := b.client.ServicesApi.DeleteLoadBalancerService(b.client.Context, id)
	statusCode := 0
	if resp != nil {
		statusCode = resp.StatusCode
	}
	return statusCode, err
}

func (b *nsxtBroker) CreateLoadBalancerVirtualServer(server loadbalancer.LbVirtualServer) (loadbalancer.LbVirtualServer, error) {
	result, _, err := b.client.ServicesApi.CreateLoadBalancerVirtualServer(b.client.Context, server)
	return result, err
}

func (b *nsxtBroker) ListLoadBalancerVirtualServers() (loadbalancer.LbVirtualServerListResult, error) {
	result, _, err := b.client.ServicesApi.ListLoadBalancerVirtualServers(b.client.Context, nil)
	return result, err
}

func (b *nsxtBroker) UpdateLoadBalancerVirtualServer(server loadbalancer.LbVirtualServer) (loadbalancer.LbVirtualServer, error) {
	result, _, err := b.client.ServicesApi.UpdateLoadBalancerVirtualServer(b.client.Context, server.Id, server)
	return result, err
}

func (b *nsxtBroker) DeleteLoadBalancerVirtualServer(id string) (int, error) {
	resp, err := b.client.ServicesApi.DeleteLoadBalancerVirtualServer(b.client.Context, id, nil)
	statusCode := 0
	if resp != nil {
		statusCode = resp.StatusCode
	}
	return statusCode, err
}

func (b *nsxtBroker) CreateLoadBalancerPool(pool loadbalancer.LbPool) (loadbalancer.LbPool, error) {
	result, _, err := b.client.ServicesApi.CreateLoadBalancerPool(b.client.Context, pool)
	return result, err
}

func (b *nsxtBroker) ReadLoadBalancerPool(id string) (loadbalancer.LbPool, error) {
	pool, _, err := b.client.ServicesApi.ReadLoadBalancerPool(b.client.Context, id)
	return pool, err
}

func (b *nsxtBroker) ListLoadBalancerPools() (loadbalancer.LbPoolListResult, error) {
	list, _, err := b.client.ServicesApi.ListLoadBalancerPools(b.client.Context, nil)
	return list, err
}

func (b *nsxtBroker) UpdateLoadBalancerPool(pool loadbalancer.LbPool) (loadbalancer.LbPool, error) {
	result, _, err := b.client.ServicesApi.UpdateLoadBalancerPool(b.client.Context, pool.Id, pool)
	return result, err
}

func (b *nsxtBroker) DeleteLoadBalancerPool(id string) (int, error) {
	resp, err := b.client.ServicesApi.DeleteLoadBalancerPool(b.client.Context, id)
	statusCode := 0
	if resp != nil {
		statusCode = resp.StatusCode
	}
	return statusCode, err
}

func (b *nsxtBroker) AllocateFromIpPool(ipPoolID string) (string, int, error) {
	allocationIPAddress, resp, err := b.client.PoolManagementApi.AllocateOrReleaseFromIpPool(b.client.Context,
		ipPoolID, manager.AllocationIpAddress{}, "ALLOCATE")
	statusCode := 0
	if resp != nil {
		statusCode = resp.StatusCode
	}
	return allocationIPAddress.AllocationId, statusCode, err
}

func (b *nsxtBroker) ListIpPoolAllocations(ipPoolID string) ([]string, int, error) {
	resultList, resp, err := b.client.PoolManagementApi.ListIpPoolAllocations(b.client.Context, ipPoolID)
	statusCode := 0
	if resp != nil {
		statusCode = resp.StatusCode
	}
	addresses := []string{}
	if err != nil {
		for _, item := range resultList.Results {
			addresses = append(addresses, item.AllocationId)
		}
	}
	return addresses, statusCode, err
}

func (b *nsxtBroker) ReleaseFromIpPool(ipPoolID, ipAddress string) (int, error) {
	allocationIpAddress := manager.AllocationIpAddress{AllocationId: ipAddress}
	_, resp, err := b.client.PoolManagementApi.AllocateOrReleaseFromIpPool(b.client.Context, ipPoolID,
		allocationIpAddress, "RELEASE")
	statusCode := 0
	if resp != nil {
		statusCode = resp.StatusCode
	}
	return statusCode, err
}