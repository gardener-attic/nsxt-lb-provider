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
	"github.com/vmware/go-vmware-nsxt/common"
	"github.com/vmware/go-vmware-nsxt/loadbalancer"
)

type Access interface {
	CreateLoadBalancerService(clusterName string) (*loadbalancer.LbService, error)
	FindFreeLoadBalancerService(clusterName string) (lbService *loadbalancer.LbService, err error)
	FindLoadBalancerServiceForVirtualServer(clusterName string, serverId string) (lbService *loadbalancer.LbService, err error)
	UpdateLoadBalancerService(lbService *loadbalancer.LbService) error
	DeleteLoadBalancerService(id string) error

	CreateVirtualServer(clusterName string, objectName ObjectName, tags TagSource, ipAddress string, mapping Mapping, poolID string) (*loadbalancer.LbVirtualServer, error)
	FindVirtualServers(clusterName string, objectName ObjectName) ([]*loadbalancer.LbVirtualServer, error)
	ListVirtualServers(clusterName string) ([]*loadbalancer.LbVirtualServer, error)
	UpdateVirtualServer(server *loadbalancer.LbVirtualServer) error
	DeleteVirtualServer(id string) error

	CreatePool(clusterName string, objectName ObjectName, mapping Mapping, members []loadbalancer.PoolMember, activeMonitorIds []string) (*loadbalancer.LbPool, error)
	GetPool(id string) (*loadbalancer.LbPool, error)
	FindPool(clusterName string, objectName ObjectName, mapping Mapping) (*loadbalancer.LbPool, error)
	FindPools(clusterName string, objectName ObjectName) ([]*loadbalancer.LbPool, error)
	ListPools(clusterName string) ([]*loadbalancer.LbPool, error)
	UpdatePool(*loadbalancer.LbPool) error
	DeletePool(id string) error

	FindIPPoolByName(poolName string) (string, error)

	AllocateExternalIPAddress(ipPoolID string) (string, error)
	IsAllocatedExternalIPAddress(ipPoolID string, address string) (bool, error)
	ReleaseExternalIPAddress(ipPoolID string, address string) error

	CreateTCPMonitor(clusterName string, objectName ObjectName, mapping Mapping) (*loadbalancer.LbTcpMonitor, error)
	FindTCPMonitors(clusterName string, objectName ObjectName) ([]*loadbalancer.LbTcpMonitor, error)
	ListTCPMonitorLight(clusterName string) ([]*loadbalancer.LbMonitor, error)
	UpdateTCPMonitor(monitor *loadbalancer.LbTcpMonitor) error
	DeleteTCPMonitor(id string) error
}

type TagSource interface {
	Tags() []common.Tag
}

type TagsSourceFunc func() []common.Tag

func (n TagsSourceFunc) Tags() []common.Tag {
	return n()
}

var EmptyTagsSource = TagsSourceFunc(func() []common.Tag { return []common.Tag{} })
