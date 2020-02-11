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

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog"
)

const (
	// SizeSmall is the NSX-T load balancer size small
	SizeSmall = "SMALL"
	// SizeMedium is the NSX-T load balancer size medium
	SizeMedium = "MEDIUM"
	// SizeLarge is the NSX-T load balancer size large
	SizeLarge = "LARGE"

	// DefaultMaxRetries is the default value for max retries
	DefaultMaxRetries = 30
	// DefaultRetryMinDelay is the default value for minimum retry delay
	DefaultRetryMinDelay = 500
	// DefaultRetryMaxDelay is the default value for maximum retry delay
	DefaultRetryMaxDelay = 5000

	// DefaultLoadBalancerClass is the default load balancer class
	DefaultLoadBalancerClass = "default"
)

// LoadBalancerSizes contains the valid size names
var LoadBalancerSizes = sets.NewString(SizeSmall, SizeMedium, SizeLarge)

// LBConfig is used to read and store information from the cloud configuration file
type LBConfig struct {
	LoadBalancer        LoadBalancerConfig                  `gcfg:"LoadBalancer"`
	LoadBalancerClasses map[string]*LoadBalancerClassConfig `gcfg:"LoadBalancerClass"`
	NSXT                NsxtConfig                          `gcfg:"NSX-T"`
	AdditionalTags      map[string]string                   `gcfg:"Tags"`
	NSXTSimulation      *NsxtSimulation                     `gcfg:"NSX-T-Simulation"`
}

// LoadBalancerConfig contains the configuration for the load balancer itself
type LoadBalancerConfig struct {
	IPPoolName      string `gcfg:"ipPoolName"`
	IPPoolID        string `gcfg:"ipPoolID"`
	Size            string `gcfg:"size"`
	LBServiceID     string `gcfg:"lbServiceId"`
	LogicalRouterID string `gcfg:"logicalRouterId"`
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
	// True if NSX-T uses self-signed cert.
	InsecureFlag       bool   `gcfg:"insecure-flag"`
	RemoteAuth         bool   `gcfg:"remote-auth"`
	MaxRetries         int    `gcfg:"max-retries"`
	RetryMinDelay      int    `gcfg:"retry-min-delay"`
	RetryMaxDelay      int    `gcfg:"retry-max-delay"`
	RetryOnStatusCodes []int  `gcfg:"retry-on-status-codes"`
	ClientAuthCertFile string `gcfg:"client-auth-cert-file"`
	ClientAuthKeyFile  string `gcfg:"client-auth-key-file"`
	CAFile             string `gcfg:"ca-file"`
}

// NsxtSimulation is a helper configuration to pass fake data for testing purposes
type NsxtSimulation struct {
	SimulatedIPPools []string `gcfg:"simulatedIPPools"`
}

// IsEnabled checks whether the load balancer feature is enabled
// It is enabled if any flavor of the load balancer configuration is given.
func (cfg *LBConfig) IsEnabled() bool {
	return len(cfg.LoadBalancerClasses) > 0 || !cfg.LoadBalancer.IsEmpty()
}

func (cfg *LBConfig) validateConfig() error {
	if cfg.LoadBalancer.LBServiceID == "" && cfg.LoadBalancer.LogicalRouterID == "" {
		msg := "load balancer servive id or logical router id required"
		klog.Errorf(msg)
		return fmt.Errorf(msg)
	}
	if !LoadBalancerSizes.Has(cfg.LoadBalancer.Size) {
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
		if cfg.LoadBalancer.IPPoolName != "" && cfg.LoadBalancer.IPPoolID != "" {
			msg := "either load balancer ipPoolName or ipPoolID can be set"
			klog.Errorf(msg)
			return fmt.Errorf(msg)
		}
	}
	if cfg.NSXTSimulation == nil {
		return cfg.NSXT.validateConfig()
	}
	return nil
}

// IsEmpty checks whether the load balancer config is empty (no values specified)
func (cfg *LoadBalancerConfig) IsEmpty() bool {
	return cfg.Size == "" && cfg.LBServiceID == "" &&
		cfg.IPPoolID == "" && cfg.IPPoolName == "" &&
		cfg.LogicalRouterID == ""
}

func (cfg *NsxtConfig) validateConfig() error {
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

// ReadConfig parses vSphere cloud config file and stores it into LBConfif.
// Environment variables are also checked
func ReadConfig(config io.Reader) (*LBConfig, error) {
	if config == nil {
		return nil, fmt.Errorf("no vSphere cloud provider config file given")
	}

	cfg := &LBConfig{}

	if err := gcfg.FatalOnly(gcfg.ReadInto(cfg, config)); err != nil {
		return nil, err
	}

	err := cfg.CompleteAndValidate()
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

// CompleteAndValidate sets default values, overrides by env and validates the resulting config
func (cfg *LBConfig) CompleteAndValidate() error {
	if !cfg.IsEnabled() {
		return nil
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
		if class.IPPoolName == "" && class.IPPoolID == "" {
			class.IPPoolID = cfg.LoadBalancer.IPPoolID
			class.IPPoolName = cfg.LoadBalancer.IPPoolName
		}
	}

	// Env Vars should override config file entries if present
	if err := cfg.NSXT.FromEnv(); err != nil {
		return err
	}

	return cfg.validateConfig()
}
