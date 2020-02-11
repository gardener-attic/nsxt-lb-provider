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
	"github.com/vmware/go-vmware-nsxt/common"
	"github.com/vmware/go-vmware-nsxt/loadbalancer"
	"k8s.io/apimachinery/pkg/types"
	clientset "k8s.io/client-go/kubernetes"
	cloudprovider "k8s.io/cloud-provider"
)

// LoadBalancer is the interface used call the load balancer functionality
// It extends the cloud controller manager LoadBalancer interface by an
// initialization function
type LoadBalancer interface {
	cloudprovider.LoadBalancer
	Initialize(client clientset.Interface, stop <-chan struct{})
}

// NSXTAccess provides methods for dealing with NSX-T objects
type NSXTAccess interface {
	// CreateLoadBalancerService creates a LbService
	CreateLoadBalancerService(clusterName string) (*loadbalancer.LbService, error)
	// FindLoadBalancerService finds a LbService by cluster name and LB service id
	FindLoadBalancerService(clusterName string, lbServiceID string) (lbService *loadbalancer.LbService, err error)
	// FindLoadBalancerServiceForVirtualServer finds the LbService for a virtual server
	FindLoadBalancerServiceForVirtualServer(clusterName string, serverID string) (lbService *loadbalancer.LbService, err error)
	// UpdateLoadBalancerService updates a LbService
	UpdateLoadBalancerService(lbService *loadbalancer.LbService) error
	// DeleteLoadBalancerService deletes a LbService by id
	DeleteLoadBalancerService(id string) error

	// CreateVirtualServer creates a virtual server
	CreateVirtualServer(clusterName string, objectName types.NamespacedName, tags TagSource, ipAddress string, mapping Mapping, poolID string) (*loadbalancer.LbVirtualServer, error)
	// FindVirtualServers finds a virtual server by cluster and object name
	FindVirtualServers(clusterName string, objectName types.NamespacedName) ([]*loadbalancer.LbVirtualServer, error)
	// ListVirtualServers finds all virtual servers for a cluster
	ListVirtualServers(clusterName string) ([]*loadbalancer.LbVirtualServer, error)
	// UpdateVirtualServer updates a virtual server
	UpdateVirtualServer(server *loadbalancer.LbVirtualServer) error
	// DeleteVirtualServer deletes a virtual server by id
	DeleteVirtualServer(id string) error

	// CreatePool creates a LbPool
	CreatePool(clusterName string, objectName types.NamespacedName, mapping Mapping, members []loadbalancer.PoolMember, activeMonitorIDs []string) (*loadbalancer.LbPool, error)
	// GetPool gets a LbPool by id
	GetPool(id string) (*loadbalancer.LbPool, error)
	// FindPool finds a LbPool for a mapping
	FindPool(clusterName string, objectName types.NamespacedName, mapping Mapping) (*loadbalancer.LbPool, error)
	// FindPools finds a LbPool by cluster and object name
	FindPools(clusterName string, objectName types.NamespacedName) ([]*loadbalancer.LbPool, error)
	// ListPools lists all LbPool for a cluster
	ListPools(clusterName string) ([]*loadbalancer.LbPool, error)
	// UpdatePool updates a LbPool
	UpdatePool(*loadbalancer.LbPool) error
	// DeletePool deletes a LbPool by id
	DeletePool(id string) error

	// FindIPPoolByName finds an IP pool by name
	FindIPPoolByName(poolName string) (string, error)

	// AllocateExternalIPAddress allocates an IP address from the given IP pool
	AllocateExternalIPAddress(ipPoolID string) (string, error)
	// IsAllocatedExternalIPAddress checks if an IP address is allocated in the given IP pool
	IsAllocatedExternalIPAddress(ipPoolID string, address string) (bool, error)
	// ReleaseExternalIPAddress releases an allocated IP address
	ReleaseExternalIPAddress(ipPoolID string, address string) error

	// CreateTCPMonitor creates a LbTcpMonitor
	CreateTCPMonitor(clusterName string, objectName types.NamespacedName, mapping Mapping) (*loadbalancer.LbTcpMonitor, error)
	// FindTCPMonitors finds a LbTcpMonitor by cluster and object name
	FindTCPMonitors(clusterName string, objectName types.NamespacedName) ([]*loadbalancer.LbTcpMonitor, error)
	// ListTCPMonitorLight list LbMonitors
	ListTCPMonitorLight(clusterName string) ([]*loadbalancer.LbMonitor, error)
	// UpdateTCPMonitor updates a LbTcpMonitor
	UpdateTCPMonitor(monitor *loadbalancer.LbTcpMonitor) error
	// DeleteTCPMonitor deletes a LbTcpMonitor by id
	DeleteTCPMonitor(id string) error
}

// TagSource is an interface to retrieve Tags
type TagSource interface {
	// Tags retrieves tags of an object
	Tags() []common.Tag
}

// TagsSourceFunc is a function type to retrieve tags
type TagsSourceFunc func() []common.Tag

// Tags implements the TagSource interface
func (n TagsSourceFunc) Tags() []common.Tag {
	return n()
}

// EmptyTagsSource is an empty tags source
var EmptyTagsSource = TagsSourceFunc(func() []common.Tag { return []common.Tag{} })
