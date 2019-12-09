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
	"sync"
)

type lbService struct {
	access      Access
	lbServiceId string
	managed     bool
	lbLock      sync.Mutex
}

func newLbService(access Access, lbServiceId string) *lbService {
	return &lbService{access: access, lbServiceId: lbServiceId, managed: lbServiceId != ""}
}

func (s *lbService) addVirtualServerToLoadBalancerService(clusterName, serverId string) error {
	s.lbLock.Lock()
	defer s.lbLock.Unlock()

	lbService, err := s.access.FindLoadBalancerService(clusterName, s.lbServiceId)
	if err != nil {
		return err
	}
	if lbService == nil {
		if s.managed {
			lbService, err = s.access.CreateLoadBalancerService(clusterName)
			if err != nil {
				return err
			}
			s.lbServiceId = lbService.Id
		} else {
			return fmt.Errorf("no more virtual servers for load balancer service")
		}
	}
	lbService.VirtualServerIds = append(lbService.VirtualServerIds, serverId)
	err = s.access.UpdateLoadBalancerService(lbService)
	if err != nil {
		return err
	}

	return nil
}

func (s *lbService) removeVirtualServerFromLoadBalancerService(clusterName, serverId string) error {
	s.lbLock.Lock()
	defer s.lbLock.Unlock()

	lbService, err := s.access.FindLoadBalancerServiceForVirtualServer(clusterName, serverId)
	if err != nil {
		return err
	}
	if lbService != nil {
		for i, id := range lbService.VirtualServerIds {
			if id == serverId {
				lbService.VirtualServerIds = append(lbService.VirtualServerIds[:i], lbService.VirtualServerIds[i+1:]...)
				break
			}
		}
		if s.managed && lbService.Id == s.lbServiceId && len(lbService.VirtualServerIds) == 0 {
			err := s.access.DeleteLoadBalancerService(lbService.Id)
			if err != nil {
				return err
			}
		} else {
			err := s.access.UpdateLoadBalancerService(lbService)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
