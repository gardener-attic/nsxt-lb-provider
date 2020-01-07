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

package config

import (
	"fmt"
	"io"
	"os"
	"strconv"

	"gopkg.in/gcfg.v1"
	"k8s.io/klog"
)

const (
	// SizeSmall is the NSX-T load balancer size small (10 virtual servers)
	SizeSmall = "SMALL"
	// SizeMedium is the NSX-T load balancer size medium (100 virtual servers)
	SizeMedium = "MEDIUM"
	// SizeLarge is the NSX-T load balancer size large (1000 virtual servers)
	SizeLarge = "LARGE"

	// DefaultMaxRetries is the default value for max retries
	DefaultMaxRetries = 30
	// DefaultRetryMinDelay is teh default value for minimum retry delay
	DefaultRetryMinDelay = 500
	// DefaultRetryMaxDelay is teh default value for maximum retry delay
	DefaultRetryMaxDelay = 5000

	// DefaultLoadBalancerClass is the default load balancer class
	DefaultLoadBalancerClass = "default"
)

// SizeToMaxVirtualServers contains the mapping from size to maximum number of virtual servers
var SizeToMaxVirtualServers = map[string]int{
	SizeSmall:  20,
	SizeMedium: 100,
	SizeLarge:  1000,
}

// Config  is used to read and store information from the cloud configuration file
type Config struct {
	LoadBalancer        LoadBalancerConfig                  `gcfg:"LoadBalancer"`
	LoadBalancerClasses map[string]*LoadBalancerClassConfig `gcfg:"LoadBalancerClass"`
	NSXT                NsxtConfig                          `gcfg:"NSX-T"`
	AdditionalTags      map[string]string                   `gcfg:"Tags"`
}

// LoadBalancerConfig contains the configuration for the load balancer itself
type LoadBalancerConfig struct {
	IPPoolName  string `gcfg:"ipPoolName"`
	IPPoolID    string `gcfg:"ipPoolID"`
	Size        string `gcfg:"size"`
	LBServiceID string `gcfg:"lbServiceId"`
}

// LoadBalancerClassConfig contains the configuration for a load balancer class
type LoadBalancerClassConfig struct {
	IPPoolName string `gcfg:"ipPoolName"`
	IPPoolID   string `gcfg:"ipPoolID"`
}

// NsxtConfig contains the NSX-T specific configuration
type NsxtConfig struct {
	// NSX-T username.
	User string `gcfg:"user"`
	// NSX-T password in clear text.
	Password string `gcfg:"password"`
	// NSX-T host.
	Host string `gcfg:"host"`
	// True if vCenter uses self-signed cert.
	InsecureFlag       bool     `gcfg:"insecure-flag"`
	RemoteAuth         bool     `gcfg:"remote_auth"`
	MaxRetries         int      `gcfg:"max_retries"`
	RetryMinDelay      int      `gcfg:"retry_min_delay"`
	RetryMaxDelay      int      `gcfg:"retry_max_delay"`
	RetryOnStatusCodes []int    `gcfg:"retry_on_status_codes"`
	ClientAuthCertFile string   `gcfg:"client_auth_cert_file"`
	ClientAuthKeyFile  string   `gcfg:"client_auth_key_file"`
	CAFile             string   `gcfg:"ca_file"`
	SimulateInMemory   bool     `gcfg:"simulateInMemory"`
	SimulatedIPPools   []string `gcfg:"simulatedIPPools"`
	LogicalRouterID    string   `gcfg:"logicalRouterId"`
}

func (cfg *Config) validateConfig() error {
	if cfg.LoadBalancer.LBServiceID == "" && cfg.NSXT.LogicalRouterID == "" {
		msg := "load balancer servive id or logical router id required"
		klog.Errorf(msg)
		return fmt.Errorf(msg)
	}
	if _, ok := SizeToMaxVirtualServers[cfg.LoadBalancer.Size]; !ok {
		msg := "load balancer size is invalid"
		klog.Errorf(msg)
		return fmt.Errorf(msg)
	}
	if cfg.LoadBalancer.IPPoolID == "" && cfg.LoadBalancer.IPPoolName == "" {
		class, ok := cfg.LoadBalancerClasses[DefaultLoadBalancerClass]
		if !ok {
			msg := "no default load balancer class defined"
			klog.Errorf(msg)
			return fmt.Errorf(msg)
		} else if class.IPPoolName == "" && class.IPPoolID == "" {
			msg := "default load balancer class: ipPoolName and ipPoolID is empty"
			klog.Errorf(msg)
			return fmt.Errorf(msg)
		}
	} else {
		if cfg.LoadBalancer.IPPoolName == "" && cfg.LoadBalancer.IPPoolID == "" {
			msg := "load balancer ipPoolName and ipPoolID is empty"
			klog.Errorf(msg)
			return fmt.Errorf(msg)
		}
	}
	return cfg.NSXT.validateConfig()
}

func (cfg *NsxtConfig) validateConfig() error {
	if cfg.SimulateInMemory {
		return nil
	}

	if cfg.User == "" {
		msg := "user is empty"
		klog.Errorf(msg)
		return fmt.Errorf(msg)
	}
	if cfg.Password == "" {
		msg := "password is empty"
		klog.Errorf(msg)
		return fmt.Errorf(msg)
	}
	if cfg.Host == "" {
		msg := "host is empty"
		klog.Errorf(msg)
		return fmt.Errorf(msg)
	}
	return nil
}

// FromEnv initializes the provided configuratoin object with values
// obtained from environment variables. If an environment variable is set
// for a property that's already initialized, the environment variable's value
// takes precedence.
func (cfg *NsxtConfig) FromEnv() error {
	if v := os.Getenv("NSXT_MANAGER_HOST"); v != "" {
		cfg.Host = v
	}
	if v := os.Getenv("NSXT_USERNAME"); v != "" {
		cfg.User = v
	}
	if v := os.Getenv("NSXT_PASSWORD"); v != "" {
		cfg.Password = v
	}
	if v := os.Getenv("NSXT_ALLOW_UNVERIFIED_SSL"); v != "" {
		InsecureFlag, err := strconv.ParseBool(v)
		if err != nil {
			klog.Errorf("Failed to parse NSXT_ALLOW_UNVERIFIED_SSL: %s", err)
			return fmt.Errorf("Failed to parse NSXT_ALLOW_UNVERIFIED_SSL: %s", err)
		}
		cfg.InsecureFlag = InsecureFlag
	}
	if v := os.Getenv("NSXT_MAX_RETRIES"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			klog.Errorf("Failed to parse NSXT_MAX_RETRIES: %s", err)
			return fmt.Errorf("Failed to parse NSXT_MAX_RETRIES: %s", err)
		}
		cfg.MaxRetries = n
	}
	if v := os.Getenv("NSXT_RETRY_MIN_DELAY"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			klog.Errorf("Failed to parse NSXT_RETRY_MIN_DELAY: %s", err)
			return fmt.Errorf("Failed to parse NSXT_RETRY_MIN_DELAY: %s", err)
		}
		cfg.RetryMinDelay = n
	}
	if v := os.Getenv("NSXT_RETRY_MAX_DELAY"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			klog.Errorf("Failed to parse NSXT_RETRY_MAX_DELAY: %s", err)
			return fmt.Errorf("Failed to parse NSXT_RETRY_MAX_DELAY: %s", err)
		}
		cfg.RetryMaxDelay = n
	}
	if v := os.Getenv("NSXT_REMOTE_AUTH"); v != "" {
		remoteAuth, err := strconv.ParseBool(v)
		if err != nil {
			klog.Errorf("Failed to parse NSXT_REMOTE_AUTH: %s", err)
			return fmt.Errorf("Failed to parse NSXT_REMOTE_AUTH: %s", err)
		}
		cfg.RemoteAuth = remoteAuth
	}
	if v := os.Getenv("NSXT_CLIENT_AUTH_CERT_FILE"); v != "" {
		cfg.ClientAuthCertFile = v
	}
	if v := os.Getenv("NSXT_CLIENT_AUTH_KEY_FILE"); v != "" {
		cfg.ClientAuthKeyFile = v
	}
	if v := os.Getenv("NSXT_CA_FILE"); v != "" {
		cfg.CAFile = v
	}

	err := cfg.validateConfig()
	if err != nil {
		return err
	}

	return nil
}

// ReadConfig parses vSphere cloud config file and stores it into VSphereConfig.
// Environment variables are also checked
func ReadConfig(config io.Reader) (*Config, error) {
	if config == nil {
		return nil, fmt.Errorf("no vSphere cloud provider config file given")
	}

	cfg := &Config{}

	if err := gcfg.FatalOnly(gcfg.ReadInto(cfg, config)); err != nil {
		return nil, err
	}
	if cfg.NSXT.MaxRetries == 0 {
		cfg.NSXT.MaxRetries = DefaultMaxRetries
	}
	if cfg.NSXT.RetryMinDelay == 0 {
		cfg.NSXT.RetryMinDelay = DefaultRetryMinDelay
	}
	if cfg.NSXT.RetryMaxDelay == 0 {
		cfg.NSXT.RetryMaxDelay = DefaultRetryMaxDelay
	}
	if cfg.LoadBalancerClasses == nil {
		cfg.LoadBalancerClasses = map[string]*LoadBalancerClassConfig{}
	}
	for _, class := range cfg.LoadBalancerClasses {
		if class.IPPoolName == "" {
			if class.IPPoolID == "" {
				class.IPPoolID = cfg.LoadBalancer.IPPoolID
				class.IPPoolName = cfg.LoadBalancer.IPPoolName
			}
		}
	}

	// Env Vars should override config file entries if present
	if err := cfg.NSXT.FromEnv(); err != nil {
		return nil, err
	}

	if err := cfg.validateConfig(); err != nil {
		return nil, err
	}

	return cfg, nil
}
