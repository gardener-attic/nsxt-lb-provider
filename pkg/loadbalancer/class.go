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

	"github.com/pkg/errors"
	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt/model"

	"github.com/gardener/nsxt-lb-provider/pkg/loadbalancer/config"
)

type loadBalancerClasses struct {
	size    string
	classes map[string]*loadBalancerClass
}

type loadBalancerClass struct {
	className  string
	ipPoolName string
	ipPoolID   string
	tags       []model.Tag
}

func setupClasses(access NSXTAccess, cfg *config.LBConfig) (*loadBalancerClasses, error) {
	if !config.LoadBalancerSizes.Has(cfg.LoadBalancer.Size) {
		return nil, fmt.Errorf("invalid load balancer size %s", cfg.LoadBalancer.Size)
	}

	lbClasses := &loadBalancerClasses{
		size:    cfg.LoadBalancer.Size,
		classes: map[string]*loadBalancerClass{},
	}

	defaultConfig := &config.LoadBalancerClassConfig{
		IPPoolName: cfg.LoadBalancer.IPPoolName,
		IPPoolID:   cfg.LoadBalancer.IPPoolID,
	}
	if defCfg, ok := cfg.LoadBalancerClasses[config.DefaultLoadBalancerClass]; ok {
		if defCfg.IPPoolID != "" || defCfg.IPPoolName != "" {
			defaultConfig = defCfg
		}
	} else {
		err := lbClasses.add(access, config.DefaultLoadBalancerClass, defaultConfig, defaultConfig)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid LoadBalancerClass %s", config.DefaultLoadBalancerClass)
		}
	}

	for name, classConfig := range cfg.LoadBalancerClasses {
		if _, ok := lbClasses.classes[name]; ok {
			return nil, fmt.Errorf("duplicate LoadBalancerClass %s", name)
		}
		err := lbClasses.add(access, name, classConfig, defaultConfig)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid LoadBalancerClass %s", name)
		}
	}

	return lbClasses, nil
}

func (c *loadBalancerClasses) GetClass(name string) *loadBalancerClass {
	return c.classes[name]
}

func (c *loadBalancerClasses) add(access NSXTAccess, name string, classConfig *config.LoadBalancerClassConfig, defaultConfig *config.LoadBalancerClassConfig) error {
	var err error
	ipPoolName := classConfig.IPPoolName
	ipPoolID := classConfig.IPPoolID
	if ipPoolID == "" && ipPoolName == "" {
		ipPoolID = defaultConfig.IPPoolID
		ipPoolName = defaultConfig.IPPoolName
	}
	if ipPoolID == "" {
		ipPoolID, err = access.FindIPPoolByName(classConfig.IPPoolName)
		if err != nil {
			return err
		}
	}
	c.classes[name] = newLBClass(name, ipPoolID, ipPoolName)
	return nil
}

func newLBClass(name, ipPoolID, ipPoolName string) *loadBalancerClass {
	tags := []model.Tag{
		{Scope: strptr(ScopeIPPoolID), Tag: strptr(ipPoolID)},
		{Scope: strptr(ScopeLBClass), Tag: strptr(name)},
	}
	return &loadBalancerClass{
		className:  name,
		ipPoolName: ipPoolName,
		ipPoolID:   ipPoolID,
		tags:       tags,
	}
}

func (c *loadBalancerClass) Tags() []model.Tag {
	return c.tags
}
