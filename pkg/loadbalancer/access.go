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
	"github.com/gardener/nsxt-lb-provider/pkg/loadbalancer/config"
	"github.com/pkg/errors"
	"github.com/vmware/go-vmware-nsxt/common"
	"github.com/vmware/go-vmware-nsxt/loadbalancer"
	"net/http"
)

const (
	// ScopeOwner is the owner scope
	ScopeOwner = "owner"
	// ScopeCluster is the cluster scope
	ScopeCluster = "cluster"
	// ScopeService is the service scope
	ScopeService = "service"
	// ScopePort is the port scope
	ScopePort = "port"
	// ScopeIPPoolID is the IP pool id scope
	ScopeIPPoolID = "ippoolid"
	// ScopeLBClass is the load balancer class scope
	ScopeLBClass = "lbclass"
)

type access struct {
	broker       NsxtBroker
	config       *config.LBConfig
	ownerTag     common.Tag
	standardTags []common.Tag
}

var _ Access = &access{}

// NewAccess creates a new Access instance
func NewAccess(broker NsxtBroker, config *config.LBConfig) (Access, error) {
	ownerTag := common.Tag{Scope: ScopeOwner, Tag: AppName}
	standardTags := []common.Tag{ownerTag}
	for k, v := range config.AdditionalTags {
		standardTags = append(standardTags, common.Tag{Scope: k, Tag: v})
	}
	return &access{
		broker:       broker,
		config:       config,
		ownerTag:     ownerTag,
		standardTags: standardTags,
	}, nil
}

func (a *access) FindIPPoolByName(poolName string) (string, error) {
	objList, err := a.broker.ListIPPools()
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
		Enabled:     true,
		Attachment: &common.ResourceReference{
			TargetId: a.config.LoadBalancer.LogicalRouterID,
		},
	}
	result, err := a.broker.CreateLoadBalancerService(lbService)
	if err != nil {
		return nil, errors.Wrapf(err, "creating load balancer service failed for cluster %s", clusterName)
	}
	return &result, nil
}

func (a *access) FindLoadBalancerService(clusterName string, id string) (*loadbalancer.LbService, error) {
	if id != "" {
		result, err := a.broker.ReadLoadBalancerService(id)
		if err != nil {
			return nil, err
		}
		if a.config.LoadBalancer.LogicalRouterID != "" && (result.Attachment == nil || result.Attachment.TargetId != a.config.LoadBalancer.LogicalRouterID) {
			targetID := "nil"
			if result.Attachment != nil {
				targetID = result.Attachment.TargetId
			}
			return nil, fmt.Errorf("load balancer service %q is configured for router %q not %q",
				result.Id,
				targetID,
				a.config.LoadBalancer.LogicalRouterID,
			)
		}
		return &result, nil
	}
	return a.findLoadBalancerService(clusterName, func(item *loadbalancer.LbService) bool {
		free := config.SizeToMaxVirtualServers[item.Size] - len(item.VirtualServerIds)
		return free > 0
	})
}

type selector func(*loadbalancer.LbService) bool

func (a *access) findLoadBalancerService(clusterName string, f selector) (*loadbalancer.LbService, error) {
	list, err := a.broker.ListLoadBalancerServices()
	if err != nil {
		return nil, errors.Wrapf(err, "listing load balancer services failed")
	}
	for _, item := range list.Results {
		if a.config.LoadBalancer.LogicalRouterID != "" && item.Attachment != nil && item.Attachment.TargetId == a.config.LoadBalancer.LogicalRouterID {
			if f(&item) {
				return &item, nil
			}
		}
		if checkTags(item.Tags, a.ownerTag, clusterTag(clusterName)) {
			if f(&item) {
				return &item, nil
			}
		}
	}
	return nil, nil
}

func (a *access) FindLoadBalancerServiceForVirtualServer(clusterName string, serverID string) (lbService *loadbalancer.LbService, err error) {
	return a.findLoadBalancerService(clusterName, func(item *loadbalancer.LbService) bool {
		for _, id := range item.VirtualServerIds {
			if id == serverID {
				return true
			}
		}
		return false
	})
}

func (a *access) UpdateLoadBalancerService(lbService *loadbalancer.LbService) error {
	_, err := a.broker.UpdateLoadBalancerService(*lbService)
	if err != nil {
		return errors.Wrapf(err, "updating load balancer service %s (%s) failed", lbService.DisplayName, lbService.Id)
	}
	return nil
}

func (a *access) DeleteLoadBalancerService(id string) error {
	statusCode, err := a.broker.DeleteLoadBalancerService(id)
	if statusCode == http.StatusNotFound {
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
	result, err := a.broker.CreateLoadBalancerVirtualServer(virtualServer)
	if err != nil {
		return nil, errors.Wrapf(err, "creating virtual server failed for %s:%s with IP address %s", clusterName, objectName, ipAddress)
	}
	return &result, nil
}

func (a *access) FindVirtualServers(clusterName string, objectName ObjectName) ([]*loadbalancer.LbVirtualServer, error) {
	return a.listVirtualServers(a.ownerTag, clusterTag(clusterName), serviceTag(objectName))
}

func (a *access) ListVirtualServers(clusterName string) ([]*loadbalancer.LbVirtualServer, error) {
	return a.listVirtualServers(a.ownerTag, clusterTag(clusterName))
}

func (a *access) listVirtualServers(tags ...common.Tag) ([]*loadbalancer.LbVirtualServer, error) {
	list, err := a.broker.ListLoadBalancerVirtualServers()
	if err != nil {
		return nil, errors.Wrapf(err, "listing virtual servers failed")
	}
	var result []*loadbalancer.LbVirtualServer
	for _, item := range list.Results {
		if checkTags(item.Tags, tags...) {
			copy := item
			result = append(result, &copy)
		}
	}
	return result, nil
}

func (a *access) UpdateVirtualServer(server *loadbalancer.LbVirtualServer) error {
	_, err := a.broker.UpdateLoadBalancerVirtualServer(*server)
	if err != nil {
		return errors.Wrapf(err, "updating load balancer virtual server %s (%s) failed", server.DisplayName, server.Id)
	}
	return nil
}

func (a *access) DeleteVirtualServer(id string) error {
	statusCode, err := a.broker.DeleteLoadBalancerVirtualServer(id)
	if statusCode == http.StatusNotFound {
		return nil
	}
	if err != nil {
		return errors.Wrapf(err, "deleting virtual server %s failed", id)
	}
	return nil
}

func (a *access) CreatePool(clusterName string, objectName ObjectName, mapping Mapping, members []loadbalancer.PoolMember, activeMonitorIds []string) (*loadbalancer.LbPool, error) {
	pool := loadbalancer.LbPool{
		Description:      fmt.Sprintf("pool for cluster %s, service %s created by %s", clusterName, objectName, AppName),
		DisplayName:      fmt.Sprintf("cluster:%s:%s", clusterName, objectName),
		Tags:             append(a.standardTags, clusterTag(clusterName), serviceTag(objectName), portTag(mapping)),
		SnatTranslation:  &loadbalancer.LbSnatTranslation{Type_: "LbSnatAutoMap"},
		Members:          members,
		ActiveMonitorIds: activeMonitorIds,
	}
	result, err := a.broker.CreateLoadBalancerPool(pool)
	if err != nil {
		return nil, errors.Wrapf(err, "creating pool failed for %s:%s", clusterName, objectName)
	}
	return &result, nil
}

func (a *access) GetPool(id string) (*loadbalancer.LbPool, error) {
	pool, err := a.broker.ReadLoadBalancerPool(id)
	if err != nil {
		return nil, err
	}
	return &pool, nil
}

func (a *access) FindPool(clusterName string, objectName ObjectName, mapping Mapping) (*loadbalancer.LbPool, error) {
	list, err := a.broker.ListLoadBalancerPools()
	if err != nil {
		return nil, errors.Wrapf(err, "listing load balancer pools failed")
	}
	for _, item := range list.Results {
		if checkTags(item.Tags, a.ownerTag, clusterTag(clusterName), serviceTag(objectName), portTag(mapping)) {
			return &item, nil
		}
	}
	return nil, nil
}

func (a *access) FindPools(clusterName string, objectName ObjectName) ([]*loadbalancer.LbPool, error) {
	return a.listPools(a.ownerTag, clusterTag(clusterName), serviceTag(objectName))
}

func (a *access) ListPools(clusterName string) ([]*loadbalancer.LbPool, error) {
	return a.listPools(a.ownerTag, clusterTag(clusterName))
}

func (a *access) listPools(tags ...common.Tag) ([]*loadbalancer.LbPool, error) {
	list, err := a.broker.ListLoadBalancerPools()
	if err != nil {
		return nil, errors.Wrapf(err, "listing pools failed")
	}
	var result []*loadbalancer.LbPool
	for _, item := range list.Results {
		if checkTags(item.Tags, tags...) {
			copy := item
			result = append(result, &copy)
		}
	}
	return result, nil
}

func (a *access) UpdatePool(pool *loadbalancer.LbPool) error {
	_, err := a.broker.UpdateLoadBalancerPool(*pool)
	if err != nil {
		return errors.Wrapf(err, "updating load balancer pool %s (%s) failed", pool.DisplayName, pool.Id)
	}
	return nil
}

func (a *access) DeletePool(id string) error {
	statusCode, err := a.broker.DeleteLoadBalancerPool(id)
	if statusCode == http.StatusNotFound {
		return nil
	}
	if err != nil {
		return errors.Wrapf(err, "deleting load balancer pool %s failed", id)
	}
	return nil
}

func (a *access) CreateTCPMonitor(clusterName string, objectName ObjectName, mapping Mapping) (*loadbalancer.LbTcpMonitor, error) {
	monitor, err := a.broker.CreateLoadBalancerTCPMonitor(loadbalancer.LbTcpMonitor{
		Description: fmt.Sprintf("tcp monitor for cluster %s, service %s, port %d created by %s",
			clusterName, objectName, mapping.NodePort, AppName),
		DisplayName: fmt.Sprintf("cluster:%s:%s:%d", clusterName, objectName, mapping.NodePort),
		Tags:        append(a.standardTags, clusterTag(clusterName), serviceTag(objectName), portTag(mapping)),
		MonitorPort: fmt.Sprintf("%d", mapping.NodePort),
	})
	if err != nil {
		return nil, errors.Wrapf(err, "creating tcp monitor failed for %s:%s:%d", clusterName, objectName, mapping.NodePort)
	}
	return &monitor, nil
}

func (a *access) GetTCPMonitor(id string) (*loadbalancer.LbTcpMonitor, error) {
	monitor, err := a.broker.ReadLoadBalancerTCPMonitor(id)
	if err != nil {
		return nil, errors.Wrapf(err, "reading tcp monitor %s failed", id)
	}
	return &monitor, nil
}

func (a *access) FindTCPMonitors(clusterName string, objectName ObjectName) ([]*loadbalancer.LbTcpMonitor, error) {
	list, err := a.broker.ListLoadBalancerMonitors()
	if err != nil {
		return nil, errors.Wrapf(err, "listing load balancer monitors failed")
	}
	result := []*loadbalancer.LbTcpMonitor{}
	for _, item := range list.Results {
		if item.ResourceType == "LbTcpMonitor" && checkTags(item.Tags, a.ownerTag, clusterTag(clusterName), serviceTag(objectName)) {
			monitor, err := a.broker.ReadLoadBalancerTCPMonitor(item.Id)
			if err != nil {
				return nil, err
			}
			result = append(result, &monitor)
		}
	}
	return result, nil
}

func (a *access) ListTCPMonitorLight(clusterName string) ([]*loadbalancer.LbMonitor, error) {
	list, err := a.broker.ListLoadBalancerMonitors()
	if err != nil {
		return nil, errors.Wrapf(err, "listing load balancer monitors failed")
	}
	result := []*loadbalancer.LbMonitor{}
	for _, item := range list.Results {
		if item.ResourceType == "LbTcpMonitor" && checkTags(item.Tags, a.ownerTag, clusterTag(clusterName)) {
			copy := item
			result = append(result, &copy)
		}
	}
	return result, nil
}

func (a *access) UpdateTCPMonitor(monitor *loadbalancer.LbTcpMonitor) error {
	_, err := a.broker.UpdateLoadBalancerTCPMonitor(*monitor)
	if err != nil {
		return errors.Wrapf(err, "updating load balancer TCP monitor %s (%s) failed", monitor.DisplayName, monitor.Id)
	}
	return nil
}

func (a *access) DeleteTCPMonitor(id string) error {
	statusCode, err := a.broker.DeleteLoadBalancerMonitor(id)
	if statusCode == http.StatusNotFound {
		return nil
	}
	if err != nil {
		return errors.Wrapf(err, "deleting monitor %s failed", id)
	}
	return nil
}

func (a *access) AllocateExternalIPAddress(ipPoolID string) (string, error) {
	ipAddress, statusCode, err := a.broker.AllocateFromIPPool(ipPoolID)
	if err != nil {
		return "", errors.Wrapf(err, "allocating external IP address failed")
	}
	if statusCode != http.StatusOK {
		return "", fmt.Errorf("allocating external IP address failed with unexpected status code %d", statusCode)
	}
	return ipAddress, nil
}

func (a *access) IsAllocatedExternalIPAddress(ipPoolID string, ipAddress string) (bool, error) {
	ipAddresses, statusCode, err := a.broker.ListIPPoolAllocations(ipPoolID)
	if statusCode == http.StatusNotFound {
		return false, nil
	}
	if err != nil {
		return false, errors.Wrapf(err, "listing IP addresses from load balancer IP pool %s (%s) failed",
			a.config.LoadBalancer.IPPoolName, ipPoolID)
	}
	if statusCode != http.StatusOK {
		return false, fmt.Errorf("unexpected status code %d returned on listing IP addresses from load balancer IP pool %s (%s)",
			statusCode, a.config.LoadBalancer.IPPoolName, ipPoolID)
	}

	for _, address := range ipAddresses {
		if address == ipAddress {
			return true, nil
		}
	}
	return false, nil
}

func (a *access) ReleaseExternalIPAddress(ipPoolID string, address string) error {
	statusCode, err := a.broker.ReleaseFromIPPool(ipPoolID, address)
	if statusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d returned releasing IP address %s from load balancer IP pool %s (%s)",
			statusCode, address, a.config.LoadBalancer.IPPoolName, ipPoolID)
	}
	if err != nil {
		return errors.Wrapf(err, "releasing external IP address %s failed", address)
	}
	return nil
}
