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
	corev1 "k8s.io/api/core/v1"

	"github.com/gardener/nsxt-lb-provider/pkg/loadbalancer/config"
)

type loadBalancerClasses struct {
	size    string
	classes map[string]*loadBalancerClass
}

type loadBalancerClass struct {
	className     string
	ipPool        Reference
	tcpAppProfile Reference
	udpAppProfile Reference

	tags []model.Tag
}

func setupClasses(access NSXTAccess, cfg *config.LBConfig) (*loadBalancerClasses, error) {
	if !config.LoadBalancerSizes.Has(cfg.LoadBalancer.Size) {
		return nil, fmt.Errorf("invalid load balancer size %s", cfg.LoadBalancer.Size)
	}

	lbClasses := &loadBalancerClasses{
		size:    cfg.LoadBalancer.Size,
		classes: map[string]*loadBalancerClass{},
	}

	defaultClass := newLBClass(config.DefaultLoadBalancerClass, &cfg.LoadBalancer.LoadBalancerClassConfig, nil)
	if defCfg, ok := cfg.LoadBalancerClasses[defaultClass.className]; ok {
		defaultClass = newLBClass(config.DefaultLoadBalancerClass, defCfg, defaultClass)
	} else {
		err := lbClasses.add(access, defaultClass)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid LoadBalancerClass %s", defaultClass.className)
		}
	}

	for name, classConfig := range cfg.LoadBalancerClasses {
		if _, ok := lbClasses.classes[name]; ok {
			return nil, fmt.Errorf("duplicate LoadBalancerClass %s", name)
		}
		class := newLBClass(name, classConfig, defaultClass)
		err := lbClasses.add(access, class)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid LoadBalancerClass %s", name)
		}
	}

	return lbClasses, nil
}

func (c *loadBalancerClasses) GetClass(name string) *loadBalancerClass {
	return c.classes[name]
}

func (c *loadBalancerClasses) add(access NSXTAccess, class *loadBalancerClass) error {
	var err error
	ipPoolID := class.ipPool.Identifier
	if ipPoolID == "" {
		ipPoolID, err = access.FindIPPoolByName(class.ipPool.Name)
		if err != nil {
			return err
		}
		class.ipPool.Identifier = ipPoolID
	}
	class.tags = []model.Tag{
		newTag(ScopeIPPoolID, ipPoolID),
		newTag(ScopeLBClass, class.className),
	}
	c.classes[class.className] = class
	return nil
}

func newLBClass(name string, classConfig *config.LoadBalancerClassConfig, defaults *loadBalancerClass) *loadBalancerClass {
	class := loadBalancerClass{
		className: name,
		ipPool: Reference{
			Identifier: classConfig.IPPoolID,
			Name:       classConfig.IPPoolName,
		},
		tcpAppProfile: Reference{
			Identifier: classConfig.TCPAppProfilePath,
			Name:       classConfig.TCPAppProfileName,
		},
		udpAppProfile: Reference{
			Identifier: classConfig.UDPAppProfilePath,
			Name:       classConfig.UDPAppProfileName,
		},
		tags: []model.Tag{},
	}
	if defaults != nil {
		if class.ipPool.IsEmpty() {
			class.ipPool = defaults.ipPool
		}
		if class.tcpAppProfile.IsEmpty() {
			class.tcpAppProfile = defaults.tcpAppProfile
		}
		if class.udpAppProfile.IsEmpty() {
			class.udpAppProfile = defaults.udpAppProfile
		}
	}
	return &class
}

func (c *loadBalancerClass) Tags() []model.Tag {
	return c.tags
}

func (c *loadBalancerClass) AppProfile(protocol corev1.Protocol) (Reference, error) {
	switch protocol {
	case corev1.ProtocolTCP:
		return c.tcpAppProfile, nil
	case corev1.ProtocolUDP:
		return c.udpAppProfile, nil
	default:
		return Reference{}, fmt.Errorf("unexpected protocol: %s", protocol)
	}
}
