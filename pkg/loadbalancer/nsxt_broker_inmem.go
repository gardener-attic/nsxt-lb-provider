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

package loadbalancer

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/vmware/go-vmware-nsxt/loadbalancer"
	"github.com/vmware/go-vmware-nsxt/manager"
)

type inmemoryNsxtBroker struct {
	lock             sync.Mutex
	ipPools          map[string]*memIPPool
	lbServices       []loadbalancer.LbService
	lbVirtualServers []loadbalancer.LbVirtualServer
	lbPools          []loadbalancer.LbPool
	lbMonitors       []loadbalancer.LbTcpMonitor

	nextID int
}

type memIPPool struct {
	id            string
	name          string
	addressPrefix string
	size          int
	allocations   map[string]struct{}
}

func newMemIPPool(id, name, addressPrefix string, size int) *memIPPool {
	return &memIPPool{
		id:            id,
		name:          name,
		addressPrefix: addressPrefix,
		size:          size,
		allocations:   map[string]struct{}{},
	}
}

func (p *memIPPool) toIPPool() manager.IpPool {
	return manager.IpPool{
		Id:          p.id,
		DisplayName: p.name,
	}
}

// NewInMemoryNsxtBroker creates a new mock in-menory NsxtBroker
func NewInMemoryNsxtBroker(ipPoolNames ...string) NsxtBroker {
	ipPools := map[string]*memIPPool{}
	for i, name := range ipPoolNames {
		id := fmt.Sprintf("ip-pool-%d", i)
		ipPools[id] = newMemIPPool(id, name, fmt.Sprintf("10.111.%d", i), 10)
	}
	return &inmemoryNsxtBroker{
		ipPools:          ipPools,
		lbServices:       []loadbalancer.LbService{},
		lbVirtualServers: []loadbalancer.LbVirtualServer{},
		lbPools:          []loadbalancer.LbPool{},
	}
}

func (b *inmemoryNsxtBroker) ListIPPools() (manager.IpPoolListResult, error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	listResult := manager.IpPoolListResult{}
	for _, ipPool := range b.ipPools {
		listResult.Results = append(listResult.Results, ipPool.toIPPool())
	}
	return listResult, nil
}

func (b *inmemoryNsxtBroker) CreateLoadBalancerService(service loadbalancer.LbService) (loadbalancer.LbService, error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	service.Id = fmt.Sprintf("lbservice-%d", b.nextID)
	b.nextID++
	b.lbServices = append(b.lbServices, service)
	return service, nil
}

func (b *inmemoryNsxtBroker) ReadLoadBalancerService(id string) (loadbalancer.LbService, error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	for _, service := range b.lbServices {
		if service.Id == id {
			return service, nil
		}
	}
	return loadbalancer.LbService{}, fmt.Errorf("LbService %s not found", id)
}

func (b *inmemoryNsxtBroker) ListLoadBalancerServices() (loadbalancer.LbServiceListResult, error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	listResult := loadbalancer.LbServiceListResult{Results: b.lbServices}
	return listResult, nil
}

func (b *inmemoryNsxtBroker) UpdateLoadBalancerService(service loadbalancer.LbService) (loadbalancer.LbService, error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	for i, svc := range b.lbServices {
		if svc.Id == service.Id {
			b.lbServices[i] = service
			return service, nil
		}
	}
	return loadbalancer.LbService{}, fmt.Errorf("LbService %s not found", service.Id)
}

func (b *inmemoryNsxtBroker) DeleteLoadBalancerService(id string) (statusCode int, err error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	for i, svc := range b.lbServices {
		if svc.Id == id {
			b.lbServices = append(b.lbServices[:i], b.lbServices[i+1:]...)
			return http.StatusOK, nil
		}
	}
	return http.StatusNotFound, fmt.Errorf("LbService %s not found", id)
}

func (b *inmemoryNsxtBroker) CreateLoadBalancerVirtualServer(server loadbalancer.LbVirtualServer) (loadbalancer.LbVirtualServer, error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	server.Id = fmt.Sprintf("server-%d", b.nextID)
	b.nextID++
	b.lbVirtualServers = append(b.lbVirtualServers, server)
	return server, nil
}

func (b *inmemoryNsxtBroker) ListLoadBalancerVirtualServers() (loadbalancer.LbVirtualServerListResult, error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	listResult := loadbalancer.LbVirtualServerListResult{Results: b.lbVirtualServers}
	return listResult, nil
}

func (b *inmemoryNsxtBroker) UpdateLoadBalancerVirtualServer(server loadbalancer.LbVirtualServer) (loadbalancer.LbVirtualServer, error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	for i, srv := range b.lbVirtualServers {
		if srv.Id == server.Id {
			b.lbVirtualServers[i] = server
			return server, nil
		}
	}
	return loadbalancer.LbVirtualServer{}, fmt.Errorf("LbVirtualServer %s not found", server.Id)
}

func (b *inmemoryNsxtBroker) DeleteLoadBalancerVirtualServer(id string) (statusCode int, err error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	for i, server := range b.lbVirtualServers {
		if server.Id == id {
			b.lbVirtualServers = append(b.lbVirtualServers[:i], b.lbVirtualServers[i+1:]...)
			return http.StatusOK, nil
		}
	}
	return http.StatusNotFound, fmt.Errorf("LbVirtualServer %s not found", id)
}

func (b *inmemoryNsxtBroker) CreateLoadBalancerPool(pool loadbalancer.LbPool) (loadbalancer.LbPool, error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	pool.Id = fmt.Sprintf("pool-%d", b.nextID)
	b.nextID++
	b.lbPools = append(b.lbPools, pool)
	return pool, nil
}

func (b *inmemoryNsxtBroker) ReadLoadBalancerPool(id string) (loadbalancer.LbPool, error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	for _, pool := range b.lbPools {
		if pool.Id == id {
			return pool, nil
		}
	}
	return loadbalancer.LbPool{}, fmt.Errorf("LbPool %s not found", id)
}

func (b *inmemoryNsxtBroker) ListLoadBalancerPools() (loadbalancer.LbPoolListResult, error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	listResult := loadbalancer.LbPoolListResult{Results: b.lbPools}
	return listResult, nil
}

func (b *inmemoryNsxtBroker) UpdateLoadBalancerPool(pool loadbalancer.LbPool) (loadbalancer.LbPool, error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	for i, p := range b.lbPools {
		if p.Id == pool.Id {
			b.lbPools[i] = pool
			return pool, nil
		}
	}
	return loadbalancer.LbPool{}, fmt.Errorf("LbPool %s not found", pool.Id)
}

func (b *inmemoryNsxtBroker) DeleteLoadBalancerPool(id string) (statusCode int, err error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	for i, pool := range b.lbPools {
		if pool.Id == id {
			b.lbPools = append(b.lbPools[:i], b.lbPools[i+1:]...)
			return http.StatusOK, nil
		}
	}
	return http.StatusNotFound, fmt.Errorf("LbPool %s not found", id)
}

func (b *inmemoryNsxtBroker) AllocateFromIPPool(ipPoolID string) (ipAddress string, statusCode int, err error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	ipPool := b.ipPools[ipPoolID]
	if ipPool == nil {
		return "", http.StatusNotFound, fmt.Errorf("IP pool %s not found", ipPoolID)
	}
	if ipPool.size == len(ipPool.allocations) {
		return "", http.StatusInternalServerError, fmt.Errorf("IP pool %s is full", ipPoolID)
	}
	for i := 1; i <= ipPool.size; i++ {
		ipAddress := fmt.Sprintf("%s.%d", ipPool.addressPrefix, i)
		if _, ok := ipPool.allocations[ipAddress]; !ok {
			ipPool.allocations[ipAddress] = struct{}{}
			return ipAddress, http.StatusOK, nil
		}
	}
	return "", http.StatusInternalServerError, fmt.Errorf("should never be reached")
}

func (b *inmemoryNsxtBroker) ListIPPoolAllocations(ipPoolID string) (ipAddresses []string, statusCode int, err error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	ipPool := b.ipPools[ipPoolID]
	if ipPool == nil {
		return nil, http.StatusNotFound, fmt.Errorf("IP pool %s not found", ipPoolID)
	}

	ipAddresses = []string{}
	for ipAddress := range ipPool.allocations {
		ipAddresses = append(ipAddresses, ipAddress)
	}
	return ipAddresses, http.StatusOK, nil
}

func (b *inmemoryNsxtBroker) ReleaseFromIPPool(ipPoolID, ipAddress string) (statusCode int, err error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	ipPool := b.ipPools[ipPoolID]
	if ipPool == nil {
		return http.StatusNotFound, fmt.Errorf("IP pool %s not found", ipPoolID)
	}
	if _, ok := ipPool.allocations[ipAddress]; !ok {
		return http.StatusNotFound, fmt.Errorf("IP address %s not found", ipAddress)
	}
	delete(ipPool.allocations, ipAddress)
	return http.StatusOK, nil
}

func (b *inmemoryNsxtBroker) CreateLoadBalancerTCPMonitor(monitor loadbalancer.LbTcpMonitor) (loadbalancer.LbTcpMonitor, error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	monitor.Id = fmt.Sprintf("tcp-monitor-%d", b.nextID)
	b.nextID++
	b.lbMonitors = append(b.lbMonitors, monitor)

	return monitor, nil
}

func (b *inmemoryNsxtBroker) ListLoadBalancerMonitors() (loadbalancer.LbMonitorListResult, error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	monitors := []loadbalancer.LbMonitor{}
	for _, monitor := range b.lbMonitors {
		plainMonitor := loadbalancer.LbMonitor{
			Description:  monitor.Description,
			DisplayName:  monitor.DisplayName,
			Id:           monitor.Id,
			ResourceType: monitor.ResourceType,
			Tags:         monitor.Tags,
		}
		monitors = append(monitors, plainMonitor)
	}
	listResult := loadbalancer.LbMonitorListResult{Results: monitors}
	return listResult, nil
}

func (b *inmemoryNsxtBroker) ReadLoadBalancerTCPMonitor(id string) (loadbalancer.LbTcpMonitor, error) {
	for _, monitor := range b.lbMonitors {
		if monitor.Id == id {
			return monitor, nil
		}
	}
	return loadbalancer.LbTcpMonitor{}, fmt.Errorf("LbTcpMonitor %s not found", id)
}

func (b *inmemoryNsxtBroker) UpdateLoadBalancerTCPMonitor(monitor loadbalancer.LbTcpMonitor) (loadbalancer.LbTcpMonitor, error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	for i, m := range b.lbMonitors {
		if m.Id == monitor.Id {
			b.lbMonitors[i] = monitor
			return monitor, nil
		}
	}
	return loadbalancer.LbTcpMonitor{}, fmt.Errorf("LbTcpMonitor %s not found", monitor.Id)
}

func (b *inmemoryNsxtBroker) DeleteLoadBalancerMonitor(id string) (int, error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	for i, monitor := range b.lbMonitors {
		if monitor.Id == id {
			b.lbMonitors = append(b.lbMonitors[:i], b.lbMonitors[i+1:]...)
			return http.StatusOK, nil
		}
	}
	return http.StatusNotFound, fmt.Errorf("LbPool %s not found", id)
}
