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
	"strings"
	"testing"
)

func TestReadConfig(t *testing.T) {
	s1 := `
[LoadBalancer]
ipPoolName = pool1
size = MEDIUM

[Tags]
tag1 = value1
tag2 = value2

[NSX-T]
user = admin
password = secret
host = nsxt-server

`
	config, err := ReadConfig(strings.NewReader(s1))
	if err != nil {
		t.Error(err)
		return
	}
	if config.LoadBalancer.IPPoolName != "pool1" {
		t.Errorf("ipPoolName %s != %s", config.LoadBalancer.IPPoolName, "pool1")
	}
	if config.LoadBalancer.Size != "MEDIUM" {
		t.Errorf("size %s != %s", config.LoadBalancer.Size, "MEDIUM")
	}
	if len(config.AdditionalTags) != 2 || config.AdditionalTags["tag1"] != "value1" || config.AdditionalTags["tag2"] != "value2" {
		t.Errorf("unexpected additionalTags %v", config.AdditionalTags)
	}
	if config.NSXT.User != "admin" {
		t.Errorf("NSX-T.user %s != %s", config.NSXT.User, "admin")
	}
	if config.NSXT.Password != "secret" {
		t.Errorf("NSX-T.password %s != %s", config.NSXT.Password, "secret")
	}
	if config.NSXT.Host != "nsxt-server" {
		t.Errorf("NSX-T.host %s != %s", config.NSXT.Host, "nsxt-server")
	}
	if config.NSXT.RetryMinDelay != DefaultRetryMinDelay || config.NSXT.RetryMaxDelay != DefaultRetryMaxDelay || config.NSXT.MaxRetries != DefaultMaxRetries {
		t.Errorf("missing default values for RetryMinDelay/RetryMaxDelay/MaxRetries")
	}
	if config.NSXT.RetryOnStatusCodes != nil {
		t.Errorf("unexpected RetryOnStatusCodes")
	}
}
