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
	"strconv"

	"github.com/vmware/go-vmware-nsxt/loadbalancer"
	corev1 "k8s.io/api/core/v1"
)

type Mapping struct {
	SourcePort int
	NodePort   int
	Protocol   corev1.Protocol
}

func NewMapping(servicePort corev1.ServicePort) Mapping {
	return Mapping{
		SourcePort: int(servicePort.Port),
		NodePort:   int(servicePort.NodePort),
		Protocol:   servicePort.Protocol,
	}
}

func (m Mapping) String() string {
	return fmt.Sprintf("%s/%d->%d", m.Protocol, m.SourcePort, m.NodePort)
}

func (m Mapping) MatchVirtualServer(server *loadbalancer.LbVirtualServer) bool {
	return server.Port == formatPort(m.SourcePort) && server.IpProtocol == string(m.Protocol)
}

func (m Mapping) MatchPool(pool *loadbalancer.LbPool) bool {
	return checkTags(pool.Tags, portTag(m))
}

func (m Mapping) MatchTCPMonitor(monitor *loadbalancer.LbTcpMonitor) bool {
	return checkTags(monitor.Tags, portTag(m))
}

func (m Mapping) MatchNodePort(server *loadbalancer.LbVirtualServer) bool {
	return server.DefaultPoolMemberPort == formatPort(m.NodePort)
}

func formatPort(port int) string {
	return strconv.FormatInt(int64(port), 10)
}
