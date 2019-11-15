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
	"github.com/vmware/go-vmware-nsxt/loadbalancer"
	corev1 "k8s.io/api/core/v1"
)

type state struct {
	access      Access
	clusterName string
	service     *corev1.Service
	servers     []*loadbalancer.LbVirtualServer
	ipAddress   string
	poolID      string
	pool        *loadbalancer.LbPool
}

func newState(clusterName string, service *corev1.Service, access Access) (*state, error) {
	var err error
	state := &state{access: access, clusterName: clusterName, service: service}
	state.servers, err = access.FindVirtualServers(clusterName, objectNameFromService(service))
	if err != nil {
		return nil, err
	}
	if len(service.Status.LoadBalancer.Ingress) > 0 {
		state.ipAddress = service.Status.LoadBalancer.Ingress[0].IP
	}
	if len(state.servers) > 0 {
		state.ipAddress = state.servers[0].IpAddress
		state.poolID = state.servers[0].PoolId
	}

	if state.poolID == "" {
		state.pool, err = access.FindPool(clusterName, objectNameFromService(service))
		if err != nil {
			return nil, err
		}
		if state.pool != nil {
			state.poolID = state.pool.Id
		}
	}

	return state, nil
}

func (s *state) getPool() (*loadbalancer.LbPool, error) {
	if s.pool == nil {
		var err error
		s.pool, err = s.access.GetPool(s.poolID)
		return s.pool, err
	}
	return s.pool, nil
}

func (s *state) initialize() error {
	var err error
	if s.poolID == "" {
		s.pool, err = s.access.CreatePool(s.clusterName, objectNameFromService(s.service))
		if err != nil {
			return err
		}
		s.poolID = s.pool.Id
	}
	if s.ipAddress == "" {
		s.ipAddress, err = s.access.AllocateExternalIPAddress()
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *state) finish() (*corev1.LoadBalancerStatus, error) {
	if len(s.service.Spec.Ports) == 0 {
		if s.ipAddress != "" {
			exists, err := s.access.IsAllocatedExternalIPAddress(s.ipAddress)
			if err != nil {
				return nil, err
			}
			if exists {
				err = s.access.ReleaseExternalIPAddress(s.ipAddress)
				if err != nil {
					return nil, err
				}
			}
		}
		if s.poolID != "" {
			err := s.access.DeletePool(s.poolID)
			if err != nil {
				return nil, err
			}
		}
		return nil, nil
	}
	return newLoadBalancerStatus(s.ipAddress), nil
}
