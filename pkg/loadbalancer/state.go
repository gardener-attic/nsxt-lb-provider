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

	"github.com/vmware/go-vmware-nsxt/loadbalancer"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog"
)

var verboseLevel = klog.Level(2)

type state struct {
	*lbService
	klog.Verbose
	clusterName string
	objectName  ObjectName
	service     *corev1.Service
	nodes       []*corev1.Node
	servers     []*loadbalancer.LbVirtualServer
	pools       []*loadbalancer.LbPool
	tcpMonitors []*loadbalancer.LbTcpMonitor
	ipAddress   string
	class       *loadBalancerClass
}

func newState(lbService *lbService, clusterName string, service *corev1.Service, nodes []*corev1.Node) *state {
	return &state{
		lbService:   lbService,
		clusterName: clusterName,
		service:     service,
		nodes:       nodes,
		objectName:  objectNameFromService(service),
		Verbose:     klog.V(verboseLevel),
	}
}

// CxtInfof logs with object name context
func (s *state) CtxInfof(format string, args ...interface{}) {
	if s.Verbose {
		s.Infof("%s: %s", s.objectName, fmt.Sprintf(format, args...))
	}
}

// Process processes a load balancer and ensures that all needed objects are existing
func (s *state) Process(class *loadBalancerClass) error {
	var err error
	s.servers, err = s.access.FindVirtualServers(s.clusterName, s.objectName)
	if err != nil {
		return err
	}
	s.pools, err = s.access.FindPools(s.clusterName, s.objectName)
	if err != nil {
		return err
	}
	s.tcpMonitors, err = s.access.FindTCPMonitors(s.clusterName, s.objectName)
	if err != nil {
		return err
	}
	if len(s.service.Status.LoadBalancer.Ingress) > 0 {
		s.ipAddress = s.service.Status.LoadBalancer.Ingress[0].IP
	}
	if len(s.servers) > 0 {
		s.ipAddress = s.servers[0].IpAddress
		className := getTag(s.servers[0].Tags, ScopeLBClass)
		ipPoolID := getTag(s.servers[0].Tags, ScopeIPPoolID)
		if class.className != className || class.ipPoolID != ipPoolID {
			class = newLBClass(className, ipPoolID, "")
		}
	}
	s.class = class

	for _, servicePort := range s.service.Spec.Ports {
		mapping := NewMapping(servicePort)

		monitor, err := s.getTCPMonitor(mapping)
		if err != nil {
			return err
		}
		pool, err := s.getPool(mapping, monitor)
		if err != nil {
			return err
		}
		_, err = s.getVirtualServer(mapping, pool.Id)
		if err != nil {
			return err
		}
	}
	validPoolIds, err := s.deleteOrphanVirtualServers()
	if err != nil {
		return err
	}
	s.CtxInfof("validPoolIds: %v", validPoolIds.List())
	validTCPMonitorIds, err := s.deleteOrphanPools(validPoolIds)
	if err != nil {
		return err
	}
	s.CtxInfof("validTCPMonitorIds: %v", validTCPMonitorIds.List())
	err = s.deleteOrphanTCPMonitors(validTCPMonitorIds)
	if err != nil {
		return err
	}
	return nil
}

func (s *state) deleteOrphanVirtualServers() (sets.String, error) {
	validPoolIds := sets.String{}
	for _, server := range s.servers {
		found := false
		for _, servicePort := range s.service.Spec.Ports {
			mapping := NewMapping(servicePort)
			if mapping.MatchVirtualServer(server) {
				validPoolIds.Insert(server.PoolId)
				found = true
				break
			}
		}
		if !found {
			err := s.deleteVirtualServer(server)
			if err != nil {
				return nil, err
			}
		}
	}
	return validPoolIds, nil
}

func (s *state) deleteOrphanPools(validPoolIds sets.String) (sets.String, error) {
	validTCPMonitorIds := sets.String{}
	for _, pool := range s.pools {
		found := false
		for _, servicePort := range s.service.Spec.Ports {
			mapping := NewMapping(servicePort)
			if mapping.MatchPool(pool) && validPoolIds.Has(pool.Id) {
				if len(pool.ActiveMonitorIds) > 0 {
					validTCPMonitorIds.Insert(pool.ActiveMonitorIds...)
				}
				found = true
				break
			}
		}
		if !found {
			err := s.deletePool(pool)
			if err != nil {
				return nil, err
			}
		}
	}
	return validTCPMonitorIds, nil
}

func (s *state) deleteOrphanTCPMonitors(validTCPMonitorIds sets.String) error {
	for _, monitor := range s.tcpMonitors {
		found := false
		for _, servicePort := range s.service.Spec.Ports {
			mapping := NewMapping(servicePort)
			if mapping.MatchTCPMonitor(monitor) && validTCPMonitorIds.Has(monitor.Id) {
				found = true
				break
			}
		}
		if !found {
			err := s.deleteTCPMonitor(monitor)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *state) allocateResources() (allocated bool, err error) {
	if s.ipAddress == "" {
		s.ipAddress, err = s.access.AllocateExternalIPAddress(s.class.ipPoolID)
		if err != nil {
			return
		}
		allocated = true
		s.CtxInfof("allocated IP address %s from pool %s", s.ipAddress, s.class.ipPoolID)
	}
	return
}

func (s *state) releaseResources() error {
	if s.ipAddress != "" {
		exists, err := s.access.IsAllocatedExternalIPAddress(s.class.ipPoolID, s.ipAddress)
		if err != nil {
			return err
		}
		if exists {
			s.CtxInfof("releasing IP address %s to pool %s", s.ipAddress, s.class.ipPoolID)
			err = s.access.ReleaseExternalIPAddress(s.class.ipPoolID, s.ipAddress)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *state) loggedReleaseResources() {
	err := s.releaseResources()
	if err != nil {
		s.CtxInfof("failed to release IP address %s to pool %s", s.ipAddress, s.class.ipPoolID)
	}
}

// Finish performs cleanup after Process
func (s *state) Finish() (*corev1.LoadBalancerStatus, error) {
	if len(s.service.Spec.Ports) == 0 {
		err := s.releaseResources()
		if err != nil {
			return nil, err
		}
		return nil, nil
	}
	return newLoadBalancerStatus(s.ipAddress), nil
}

func (s *state) getTCPMonitor(mapping Mapping) (*loadbalancer.LbTcpMonitor, error) {
	if mapping.Protocol == corev1.ProtocolTCP {
		for _, m := range s.tcpMonitors {
			if mapping.MatchTCPMonitor(m) {
				err := s.updateTCPMonitor(m, mapping)
				if err != nil {
					return nil, err
				}
				return m, nil
			}
		}
		return s.createTCPMonitor(mapping)
	}
	return nil, nil
}

func (s *state) createTCPMonitor(mapping Mapping) (*loadbalancer.LbTcpMonitor, error) {
	monitor, err := s.access.CreateTCPMonitor(s.clusterName, s.objectName, mapping)
	if err == nil {
		s.CtxInfof("created LbTcpMonitor %s for %s", monitor.Id, mapping)
		s.tcpMonitors = append(s.tcpMonitors, monitor)
	}
	return monitor, err
}

func (s *state) updateTCPMonitor(monitor *loadbalancer.LbTcpMonitor, mapping Mapping) error {
	if monitor.MonitorPort == fmt.Sprintf("%d", mapping.NodePort) {
		return nil
	}
	monitor.MonitorPort = fmt.Sprintf("%d", mapping.NodePort)
	s.CtxInfof("updating LbTcpMonitor %s for %s", monitor.Id, mapping)
	return s.access.UpdateTCPMonitor(monitor)
}

func (s *state) deleteTCPMonitor(monitor *loadbalancer.LbTcpMonitor) error {
	s.CtxInfof("deleting LbTcpMonitor %s for %s", monitor.Id, getTag(monitor.Tags, ScopePort))
	return s.access.DeleteTCPMonitor(monitor.Id)
}

func (s *state) getPool(mapping Mapping, monitor *loadbalancer.LbTcpMonitor) (*loadbalancer.LbPool, error) {
	var activeMonitorIds []string
	if monitor != nil {
		activeMonitorIds = []string{monitor.Id}
	}
	for _, pool := range s.pools {
		if mapping.MatchPool(pool) {
			err := s.updatePool(pool, mapping, activeMonitorIds)
			return pool, err
		}
	}
	return s.createPool(mapping, activeMonitorIds)
}

func (s *state) createPool(mapping Mapping, activeMonitorIds []string) (*loadbalancer.LbPool, error) {
	members, _ := s.updatedPoolMembers(nil)
	pool, err := s.access.CreatePool(s.clusterName, s.objectName, mapping, members, activeMonitorIds)
	if err == nil {
		s.CtxInfof("created LbPool %s for %s", pool.Id, mapping)
		s.pools = append(s.pools, pool)
	}
	return pool, err
}

func (s *state) UpdatePoolMembers() error {
	pools, err := s.access.FindPools(s.clusterName, s.objectName)
	if err != nil {
		return err
	}
	for _, servicePort := range s.service.Spec.Ports {
		mapping := NewMapping(servicePort)
		for _, pool := range pools {
			if mapping.MatchPool(pool) {
				err = s.updatePool(pool, mapping, pool.ActiveMonitorIds)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (s *state) updatePool(pool *loadbalancer.LbPool, mapping Mapping, activeMonitorIds []string) error {
	newMembers, modified := s.updatedPoolMembers(pool.Members)
	if modified || !stringsEquals(activeMonitorIds, pool.ActiveMonitorIds) {
		pool.Members = newMembers
		pool.ActiveMonitorIds = activeMonitorIds
		s.CtxInfof("updating LbPool %s for %s, #members=%d", pool.Id, mapping, len(pool.Members))
		err := s.access.UpdatePool(pool)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *state) updatedPoolMembers(oldMembers []loadbalancer.PoolMember) ([]loadbalancer.PoolMember, bool) {
	modified := false
	nodeIPAddresses := collectNodeInternalAddresses(s.nodes)
	newMembers := []loadbalancer.PoolMember{}
	for _, member := range oldMembers {
		if _, ok := nodeIPAddresses[member.IpAddress]; ok {
			newMembers = append(newMembers, member)
		} else {
			modified = true
		}
	}
	if len(nodeIPAddresses) > len(newMembers) {
		for nodeIPAddress, nodeName := range nodeIPAddresses {
			found := false
			for _, member := range oldMembers {
				if member.IpAddress == nodeIPAddress {
					found = true
					break
				}
			}
			if !found {
				member := loadbalancer.PoolMember{
					AdminState:  "ENABLED",
					DisplayName: fmt.Sprintf("%s:%s", s.clusterName, nodeName),
					IpAddress:   nodeIPAddress,
				}
				newMembers = append(newMembers, member)
				modified = true
			}
		}
	}
	return newMembers, modified
}

func (s *state) deletePool(pool *loadbalancer.LbPool) error {
	s.CtxInfof("deleting LbPool %s for %s", pool.Id, getTag(pool.Tags, ScopePort))
	return s.access.DeletePool(pool.Id)
}

func (s *state) getVirtualServer(mapping Mapping, poolID string) (*loadbalancer.LbVirtualServer, error) {
	for _, server := range s.servers {
		if mapping.MatchVirtualServer(server) {
			err := s.updateVirtualServer(server, mapping, poolID)
			if err != nil {
				return nil, err
			}
			return server, nil
		}
	}

	return s.createVirtualServer(mapping, poolID)
}

func (s *state) createVirtualServer(mapping Mapping, poolID string) (*loadbalancer.LbVirtualServer, error) {
	allocated, err := s.allocateResources()
	if err != nil {
		return nil, err
	}

	server, err := s.access.CreateVirtualServer(s.clusterName, s.objectName, s.class, s.ipAddress, mapping, poolID)
	if err != nil {
		if allocated {
			s.loggedReleaseResources()
		}
		return nil, err
	}
	s.CtxInfof("created LbVirtualServer %s for %s", server.Id, mapping)
	s.servers = append(s.servers, server)
	err = s.lbService.addVirtualServerToLoadBalancerService(s.clusterName, server.Id)
	if err != nil {
		_ = s.access.DeleteVirtualServer(server.Id)
		if allocated {
			s.loggedReleaseResources()
		}
		return nil, err
	}
	return server, nil
}

func (s *state) updateVirtualServer(server *loadbalancer.LbVirtualServer, mapping Mapping, poolID string) error {
	if !mapping.MatchNodePort(server) || poolID != server.PoolId {
		server.DefaultPoolMemberPort = formatPort(mapping.NodePort)
		server.PoolId = poolID
		s.CtxInfof("updating LbVirtualServer %s for %s", server.Id, mapping)
		err := s.access.UpdateVirtualServer(server)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *state) deleteVirtualServer(server *loadbalancer.LbVirtualServer) error {
	err := s.lbService.removeVirtualServerFromLoadBalancerService(s.clusterName, server.Id)
	if err != nil {
		return err
	}
	s.CtxInfof("deleting LbVirtualServer %s for %s->%s", server.Id, getTag(server.Tags, ScopePort), server.DefaultPoolMemberPort)
	return s.access.DeleteVirtualServer(server.Id)
}
