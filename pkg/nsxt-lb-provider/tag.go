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

import "github.com/vmware/go-vmware-nsxt/common"

func clusterTag(clusterName string) common.Tag {
	return common.Tag{Scope: ScopeCluster, Tag: clusterName}
}

func serviceTag(objectName ObjectName) common.Tag {
	return common.Tag{Scope: ScopeService, Tag: objectName.String()}
}

func checkTags(tags []common.Tag, required ...common.Tag) bool {
outer:
	for _, req := range required {
		for _, tag := range tags {
			if tag.Scope == req.Scope {
				if tag.Tag != req.Tag {
					return false
				}
				continue outer
			}
		}
		return false
	}
	return true
}

func getTag(tags []common.Tag, scope string) string {
	for _, tag := range tags {
		if tag.Scope == scope {
			return tag.Tag
		}
	}
	return ""
}

func mapToTags(tagMap map[string]string) []common.Tag {
	tags := []common.Tag{}
	for scope, tag := range tagMap {
		tags = append(tags, common.Tag{Scope: scope, Tag: tag})
	}
	return tags
}
