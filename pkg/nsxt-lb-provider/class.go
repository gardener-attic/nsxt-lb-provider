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
	"github.com/vmware/go-vmware-nsxt/common"
)

type loadBalancerClass struct {
	className         string
	ipPoolName        string
	ipPoolID          string
	size              string
	maxVirtualServers int
	tags              []common.Tag
}

func setupClasses(access Access, cfg *config.Config) (map[string]*loadBalancerClass, error) {
	lbClasses := map[string]*loadBalancerClass{}

	if cfg.LoadBalancer != nil {
		lbClass, err := newLbClass(access, config.DefaultLoadBalancerClass, cfg.LoadBalancer)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid default LoadBalancerClass")
		}
		lbClasses[config.DefaultLoadBalancerClass] = lbClass
	}
	for name, classConfig := range cfg.LoadBalancerClass {
		if _, ok := lbClasses[name]; ok {
			return nil, fmt.Errorf("duplicate LoadBalancerClass %s", name)
		}
		lbClass, err := newLbClass(access, name, classConfig)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid LoadBalancerClass %s", name)
		}
		lbClasses[name] = lbClass
	}
	return lbClasses, nil
}

func newLbClass(access Access, name string, classConfig *config.LoadBalancerConfig) (*loadBalancerClass, error) {
	var err error
	ipPoolID := classConfig.IPPoolID
	if ipPoolID == "" {
		ipPoolID, err = access.FindIPPoolByName(classConfig.IPPoolName)
		if err != nil {
			return nil, err
		}
	}
	max, ok := config.SizeToMaxVirtualServers[classConfig.Size]
	if !ok {
		return nil, fmt.Errorf("invalid load balancer size %s", classConfig.Size)
	}
	tags := []common.Tag{
		{Scope: ScopeIPPoolID, Tag: ipPoolID},
		{Scope: ScopeLBClass, Tag: name},
	}
	return &loadBalancerClass{
		className:         name,
		ipPoolName:        classConfig.IPPoolName,
		ipPoolID:          ipPoolID,
		size:              classConfig.Size,
		maxVirtualServers: max,
		tags:              tags,
	}, nil
}

func (c *loadBalancerClass) Tags() []common.Tag {
	return c.tags
}
