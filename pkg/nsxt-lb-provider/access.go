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
	"fmt"
	"github.com/gardener/nsxt-lb-provider/pkg/nsxt-lb-provider/config"
	"github.com/pkg/errors"
	nsxt "github.com/vmware/go-vmware-nsxt"
	"github.com/vmware/go-vmware-nsxt/common"
	"github.com/vmware/go-vmware-nsxt/loadbalancer"
	"github.com/vmware/go-vmware-nsxt/manager"
	"net/http"
)

const (
	ScopeOwner    = "owner"
	ScopeCluster  = "cluster"
	ScopeService  = "service"
	ScopeIPPoolID = "ippoolid"
	ScopeLBClass  = "lbclass"
)

type access struct {
	nsxClient    *nsxt.APIClient
	config       *config.Config
	standardTags []common.Tag
}

var (
	ownerTag = common.Tag{Scope: ScopeOwner, Tag: AppName}
)

var _ Access = &access{}

func NewAccess(client *nsxt.APIClient, config *config.Config) (Access, error) {
	standardTags := []common.Tag{ownerTag}
	for k, v := range config.AdditionalTags {
		standardTags = append(standardTags, common.Tag{Scope: k, Tag: v})
	}
	return &access{
		nsxClient:    client,
		config:       config,
		standardTags: standardTags,
	}, nil
}

func (a *access) FindIPPoolByName(poolName string) (string, error) {
	objList, _, err := a.nsxClient.PoolManagementApi.ListIpPools(a.nsxClient.Context, nil)
	if err != nil {
		return "", errors.Wrap(err, "listing IP pools failed")
	}
	for _, item := range objList.Results {
		if item.DisplayName == poolName {
			return item.Id, nil
		}
	}
	return "", fmt.Errorf("load balancer IP pool named %s not found", poolName)
}

func (a *access) CreateLoadBalancerService(clusterName string) (*loadbalancer.LbService, error) {
	lbService := loadbalancer.LbService{
		Description: fmt.Sprintf("virtual server pool for cluster %s created by %s", clusterName, AppName),
		DisplayName: fmt.Sprintf("cluster:%s", clusterName),
		Tags:        append(a.standardTags, clusterTag(clusterName)),
		Size:        a.config.LoadBalancer.Size,
	}
	result, _, err := a.nsxClient.ServicesApi.CreateLoadBalancerService(a.nsxClient.Context, lbService)
	if err != nil {
		return nil, errors.Wrapf(err, "creating load balancer service failed for cluster %s", clusterName)
	}
	return &result, nil
}

func (a *access) FindFreeLoadBalancerService(clusterName string) (*loadbalancer.LbService, error) {
	return a.findLoadBalancerService(clusterName, func(item *loadbalancer.LbService) bool {
		free := config.SizeToMaxVirtualServers[item.Size] - len(item.VirtualServerIds)
		return free > 0
	})
}

type selector func(*loadbalancer.LbService) bool

func (a *access) findLoadBalancerService(clusterName string, f selector) (*loadbalancer.LbService, error) {
	list, _, err := a.nsxClient.ServicesApi.ListLoadBalancerServices(a.nsxClient.Context, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "listing load balancer services failed")
	}
	for _, item := range list.Results {
		if checkTags(item.Tags, ownerTag, clusterTag(clusterName)) && f(&item) {
			return &item, nil
		}
	}
	return nil, nil
}

func (a *access) FindLoadBalancerServiceForVirtualServer(clusterName string, serverId string) (lbService *loadbalancer.LbService, err error) {
	return a.findLoadBalancerService(clusterName, func(item *loadbalancer.LbService) bool {
		for _, id := range item.VirtualServerIds {
			if id == serverId {
				return true
			}
		}
		return false
	})
}

func (a *access) UpdateLoadBalancerService(lbService *loadbalancer.LbService) error {
	_, _, err := a.nsxClient.ServicesApi.UpdateLoadBalancerService(a.nsxClient.Context, lbService.Id, *lbService)
	if err != nil {
		return errors.Wrapf(err, "updating load balancer service %s (%s) failed", lbService.DisplayName, lbService.Id)
	}
	return nil
}

func (a *access) DeleteLoadBalancerService(id string) error {
	resp, err := a.nsxClient.ServicesApi.DeleteLoadBalancerService(a.nsxClient.Context, id)
	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if err != nil {
		return errors.Wrapf(err, "deleting load balancer service %s failed", id)
	}
	return nil
}

func (a *access) CreateVirtualServer(clusterName string, objectName ObjectName, tags TagSource, ipAddress string, mapping Mapping, poolID string) (*loadbalancer.LbVirtualServer, error) {
	virtualServer := loadbalancer.LbVirtualServer{
		Description: fmt.Sprintf("virtual server for cluster %s, service %s created by %s",
			clusterName, objectName, AppName),
		DisplayName:           fmt.Sprintf("cluster:%s:%s", clusterName, objectName),
		Tags:                  append(append(a.standardTags, clusterTag(clusterName), serviceTag(objectName)), tags.Tags()...),
		DefaultPoolMemberPort: fmt.Sprintf("%d", mapping.NodePort),
		Enabled:               true,
		IpAddress:             ipAddress,
		IpProtocol:            string(mapping.Protocol),
		PoolId:                poolID,
		Port:                  fmt.Sprintf("%d", mapping.SourcePort),
	}
	result, _, err := a.nsxClient.ServicesApi.CreateLoadBalancerVirtualServer(a.nsxClient.Context, virtualServer)
	if err != nil {
		return nil, errors.Wrapf(err, "creating virtual server failed for %s:%s with IP address %s", clusterName, objectName, ipAddress)
	}
	return &result, nil
}

func (a *access) FindVirtualServers(clusterName string, objectName ObjectName) ([]*loadbalancer.LbVirtualServer, error) {
	list, _, err := a.nsxClient.ServicesApi.ListLoadBalancerVirtualServers(a.nsxClient.Context, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "listing virtual servers failed")
	}
	var result []*loadbalancer.LbVirtualServer
	for _, item := range list.Results {
		if checkTags(item.Tags, ownerTag, clusterTag(clusterName), serviceTag(objectName)) {
			result = append(result, &item)
		}
	}
	return result, nil
}

func (a *access) ListVirtualServers(clusterName string) ([]*loadbalancer.LbVirtualServer, error) {
	list, _, err := a.nsxClient.ServicesApi.ListLoadBalancerVirtualServers(a.nsxClient.Context, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "listing virtual servers failed")
	}
	var result []*loadbalancer.LbVirtualServer
	for _, item := range list.Results {
		if checkTags(item.Tags, ownerTag, clusterTag(clusterName)) {
			result = append(result, &item)
		}
	}
	return result, nil
}

func (a *access) UpdateVirtualServer(server *loadbalancer.LbVirtualServer) error {
	_, _, err := a.nsxClient.ServicesApi.UpdateLoadBalancerVirtualServer(a.nsxClient.Context, server.Id, *server)
	if err != nil {
		return errors.Wrapf(err, "updating load balancer virtual server %s (%s) failed", server.DisplayName, server.Id)
	}
	return nil
}

func (a *access) DeleteVirtualServer(id string) error {
	resp, err := a.nsxClient.ServicesApi.DeleteLoadBalancerVirtualServer(a.nsxClient.Context, id, nil)
	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if err != nil {
		return errors.Wrapf(err, "deleting virtual server %s failed", id)
	}
	return nil
}

func (a *access) CreatePool(clusterName string, objectName ObjectName) (*loadbalancer.LbPool, error) {
	pool := loadbalancer.LbPool{
		Description: fmt.Sprintf("pool for cluster %s, service %s created by %s",
			clusterName, objectName, AppName),
		DisplayName: fmt.Sprintf("cluster:%s:%s", clusterName, objectName),
		Tags:        append(a.standardTags, clusterTag(clusterName), serviceTag(objectName)),
		Members:     []loadbalancer.PoolMember{},
	}
	result, _, err := a.nsxClient.ServicesApi.CreateLoadBalancerPool(a.nsxClient.Context, pool)
	if err != nil {
		return nil, errors.Wrapf(err, "creating pool failed for %s:%s", clusterName, objectName)
	}
	return &result, nil
}

func (a *access) GetPool(id string) (*loadbalancer.LbPool, error) {
	pool, _, err := a.nsxClient.ServicesApi.ReadLoadBalancerPool(a.nsxClient.Context, id)
	if err != nil {
		return nil, err
	}
	return &pool, nil
}

func (a *access) FindPool(clusterName string, objectName ObjectName) (*loadbalancer.LbPool, error) {
	list, _, err := a.nsxClient.ServicesApi.ListLoadBalancerPools(a.nsxClient.Context, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "listing load balancer pools failed")
	}
	for _, item := range list.Results {
		if checkTags(item.Tags, ownerTag, clusterTag(clusterName), serviceTag(objectName)) {
			return &item, nil
		}
	}
	return nil, nil
}

func (a *access) ListPools(clusterName string) ([]*loadbalancer.LbPool, error) {
	list, _, err := a.nsxClient.ServicesApi.ListLoadBalancerPools(a.nsxClient.Context, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "listing pools failed")
	}
	var result []*loadbalancer.LbPool
	for _, item := range list.Results {
		if checkTags(item.Tags, ownerTag, clusterTag(clusterName)) {
			result = append(result, &item)
		}
	}
	return result, nil
}

func (a *access) UpdatePool(pool *loadbalancer.LbPool) error {
	_, _, err := a.nsxClient.ServicesApi.UpdateLoadBalancerPool(a.nsxClient.Context, pool.Id, *pool)
	if err != nil {
		return errors.Wrapf(err, "updating load balancer pool %s (%s) failed", pool.DisplayName, pool.Id)
	}
	return nil
}

func (a *access) DeletePool(id string) error {
	resp, err := a.nsxClient.ServicesApi.DeleteLoadBalancerPool(a.nsxClient.Context, id)
	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if err != nil {
		return errors.Wrapf(err, "deleting oad balancer pool %s failed", id)
	}
	return nil
}

func (a *access) AllocateExternalIPAddress(ipPoolID string) (string, error) {
	allocationIPAddress, resp, err := a.nsxClient.PoolManagementApi.AllocateOrReleaseFromIpPool(a.nsxClient.Context,
		ipPoolID, manager.AllocationIpAddress{}, "ALLOCATE")
	if err != nil {
		return "", errors.Wrapf(err, "allocating external IP address failed")
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("allocating external IP address failed with unexpected status code %d", resp.StatusCode)
	}
	return allocationIPAddress.AllocationId, nil
}

func (a *access) IsAllocatedExternalIPAddress(ipPoolID string, ipAddress string) (bool, error) {
	resultList, resp, err := a.nsxClient.PoolManagementApi.ListIpPoolAllocations(a.nsxClient.Context, ipPoolID)
	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	if err != nil {
		return false, errors.Wrapf(err, "listing IP addresses from load balancer IP pool %s (%s) failed",
			a.config.LoadBalancer.IPPoolName, ipPoolID)
	}
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("unexpected status code %d returned on listing IP addresses from load balancer IP pool %s (%s)",
			resp.StatusCode, a.config.LoadBalancer.IPPoolName, ipPoolID)
	}

	for _, address := range resultList.Results {
		if address.AllocationId == ipAddress {
			return true, nil
		}
	}
	return false, nil
}

func (a *access) ReleaseExternalIPAddress(ipPoolID string, address string) error {
	allocationIpAddress := manager.AllocationIpAddress{AllocationId: address}
	_, resp, err := a.nsxClient.PoolManagementApi.AllocateOrReleaseFromIpPool(a.nsxClient.Context, ipPoolID,
		allocationIpAddress, "RELEASE")
	if resp != nil && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d returned releasing IP address %s from load balancer IP pool %s (%s)",
			resp.StatusCode, address, a.config.LoadBalancer.IPPoolName, ipPoolID)
	}
	if err != nil {
		return errors.Wrapf(err, "releasing external IP address %s failed", address)
	}
	return nil
}
