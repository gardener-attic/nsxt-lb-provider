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
	cloudprovider "k8s.io/cloud-provider"

	"github.com/gardener/nsxt-lb-provider/pkg/loadbalancer/config"
)

type provider struct {
	lbProvider *lbProvider
}

var _ cloudprovider.Interface = &provider{}

func newProvider(config *config.LBConfig) (*provider, error) {
	lbProvider, err := newLBProvider(config)
	if err != nil {
		return nil, err
	}

	return &provider{lbProvider: lbProvider}, nil
}

// Initialize provides the cloud with a kubernetes client builder and may spawn goroutines
// to perform housekeeping or run custom controllers specific to the cloud provider.
// Any tasks started here should be cleaned up when the stop channel closes.
func (p *provider) Initialize(clientBuilder cloudprovider.ControllerClientBuilder, stop <-chan struct{}) {
	p.lbProvider.Initialize(clientBuilder, stop)
}

// LoadBalancer returns a balancer interface. Also returns true if the interface is supported, false otherwise.
func (p *provider) LoadBalancer() (cloudprovider.LoadBalancer, bool) {
	return p.lbProvider, true
}

// Instances returns an instances interface. Also returns true if the interface is supported, false otherwise.
func (p *provider) Instances() (cloudprovider.Instances, bool) {
	return nil, false
}

// Zones returns a zones interface. Also returns true if the interface is supported, false otherwise.
func (p *provider) Zones() (cloudprovider.Zones, bool) {
	return nil, false
}

// Clusters returns a clusters interface.  Also returns true if the interface is supported, false otherwise.
func (p *provider) Clusters() (cloudprovider.Clusters, bool) {
	return nil, false
}

// Routes returns a routes interface along with whether the interface is supported.
func (p *provider) Routes() (cloudprovider.Routes, bool) {
	return nil, false
}

// ProviderName returns the cloud provider ID.
func (p *provider) ProviderName() string {
	return ProviderName
}

// HasClusterID returns true if a ClusterID is required and set
func (p *provider) HasClusterID() bool {
	return true
}
