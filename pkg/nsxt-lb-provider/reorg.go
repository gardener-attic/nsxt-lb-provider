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
	"context"
	"github.com/vmware/go-vmware-nsxt/common"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type objectName struct {
	Namespace string
	Name      string
}

func objectNameFromService(service *corev1.Service) objectName {
	return objectName{Namespace: service.Namespace, Name: service.Name}
}

func (o objectName) String() string {
	return o.Namespace + "/" + o.Name
}

func parseObjectName(name string) objectName {
	parts := strings.Split(name, "/")
	return objectName{Namespace: parts[0], Name: parts[1]}
}

const maxPeriod = 30 * time.Minute

func (p *lbProvider) reorg(client clientcorev1.ServiceInterface, stop <-chan struct{}) {
	timer := time.NewTimer(1 * time.Second)
	next := maxPeriod
	lastErrNext := 0 * time.Second
	for {
		select {
		case <-stop:
			return
		case <-timer.C:
			err := p.doReorgStep(client)
			if err == nil {
				next = maxPeriod
				lastErrNext = 0
			} else {
				if lastErrNext == 0 {
					lastErrNext = 500 * time.Millisecond
				} else {
					lastErrNext = 5 * lastErrNext / 4
					if lastErrNext > maxPeriod {
						lastErrNext = maxPeriod
					}
				}
				next = lastErrNext
			}
			timer.Reset(next)
		}
	}
}

func (p *lbProvider) doReorgStep(client clientcorev1.ServiceInterface) error {
	list, err := client.List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	services := map[objectName]struct{}{}
	for _, item := range list.Items {
		if item.Spec.Type == corev1.ServiceTypeLoadBalancer {
			services[objectNameFromService(&item)] = struct{}{}
		}
	}

	lbs := map[objectName]struct{}{}
	servers, err := p.access.ListVirtualServers(ClusterName)
	if err != nil {
		return err
	}
	for _, server := range servers {
		tag := getTag(server.Tags, ScopeService)
		if tag != "" {
			lbs[parseObjectName(tag)] = struct{}{}
		}
	}

	pools, err := p.access.ListPools(ClusterName)
	if err != nil {
		return err
	}
	for _, pool := range pools {
		tag := getTag(pool.Tags, ScopeService)
		if tag != "" {
			lbs[parseObjectName(tag)] = struct{}{}
		}
	}

	for lb := range lbs {
		if _, ok := services[lb]; !ok {
			service := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: lb.Namespace,
					Name:      lb.Name,
				},
			}
			err = p.EnsureLoadBalancerDeleted(context.TODO(), ClusterName, service)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func getTag(tags []common.Tag, scope string) string {
	for _, tag := range tags {
		if tag.Scope == scope {
			return tag.Tag
		}
	}
	return ""
}
