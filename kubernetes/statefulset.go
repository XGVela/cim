// Copyright 2020 Mavenir
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kubernetes

import (
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//GetStatefulSets get StatefulSets details
func (c *KubeClient) GetStatefulSets(namespace, statefulsetName string) (*v1.StatefulSet, error) {

	statefulsetDetails, err := c.Client.AppsV1().StatefulSets(namespace).Get(statefulsetName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return statefulsetDetails, nil
}
