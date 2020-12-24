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
	"cim/logs"
	t "cim/types"
	"cim/utility"
	"encoding/json"
	"fmt"
	"os"

	// "strings"

	"strings"
	"sync"
	"time"

	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var (
	k8sWatchLock    = new(sync.Mutex)
	containerStates = make(map[string]time.Time)
	//KubeConnect k8s connection object
	KubeConnect *KubeClient
)

type patchStringValue struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value"`
}

type ControlIpRequest struct {
	Msuid           string `json:"msuid"`
	ServiceEndpoint string `json:"service-endpoint"`
}

type ControlIpResponse struct {
	ControlIpList []string `json:"control_ip_list"`
	Error         string   `json:"error,omitempty"`
}

//GetLabelValue gets the value of the label passed, for current pod
func (c *KubeClient) GetLabelValue(label string) string {
	podDetails, err := c.Client.CoreV1().Pods(t.Namespace).Get(t.PodID, metav1.GetOptions{})
	if err != nil {
		return ""
	}
	m := podDetails.GetLabels()
	if v, ok := m[label]; ok {
		return v
	}
	return ""
}

//GetPod get pod details
func (c *KubeClient) GetPod(namespace, podName string) (*v1.Pod, error) {
	podDetails, err := c.Client.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return podDetails, nil
}

//GetPodUUID fetches the UUID of the pod
func (c *KubeClient) GetPodUUID() string {
	podDetails, err := c.Client.CoreV1().Pods(t.Namespace).Get(t.PodID, metav1.GetOptions{})
	if err != nil {
		return ""
	}

	return string(podDetails.GetUID())
}

// GetPodDeletionGracePeriod returns grace period for pod deletion in seconds
func (c *KubeClient) GetPodDeletionGracePeriod(namespace, podName string) (int64, error) {
	podDetails, err := c.Client.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})
	if err != nil {
		return -1, err
	}
	gracePeriod := podDetails.GetDeletionGracePeriodSeconds()
	if gracePeriod == nil {
		// this may happen in case the pod has not come up yet and grace period is not yet
		// populated.
		// return 0 as we may need to flush ASAP
		return 0, nil
	}
	return *gracePeriod, nil
}

//GetPod list pods details
func (c *KubeClient) ListPods(namespace string) (*v1.PodList, error) {
	podDetails, err := c.Client.CoreV1().Pods(namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return podDetails, nil
}

//Get Local daemonset Logging pod
func (c *KubeClient) GetLoggingPod(nodename string) (*v1.PodList, error) {
	pods, err := c.Client.CoreV1().Pods("").List(metav1.ListOptions{
		FieldSelector: "spec.nodeName=" + nodename,
		LabelSelector: "k8s-app=" + "fluent-bit",
	})
	if err != nil {
		return nil, err
	}
	return pods, nil
}

// AddAnnotation adds an annotation specified by path and it's value specified in valueJson, in a pod in a given namespace
func (c *KubeClient) AddAnnotation(path, valueJSON, namespace, podName string) error {
	patchData := map[string]interface{}{
		"metadata": map[string]map[string]string{
			"annotations": {
				path: valueJSON,
			},
		},
	}

	payloadBytes, err := json.Marshal(patchData)
	if err != nil {
		logs.LogConf.Error("AddAnnotation: error while marshalling the annotation data", err)
		return err
	}

	logs.LogConf.Debug("AddAnnotation: payload = ", string(payloadBytes))
	_, updateErr := c.Client.CoreV1().Pods(namespace).Patch(podName, types.StrategicMergePatchType, payloadBytes)
	if updateErr != nil {
		logs.LogConf.Error("AddAnnotation: Error while adding annotations:", updateErr)
		return updateErr
	}

	logs.LogConf.Info(fmt.Sprintf("AddAnnotation: Pod %s annotated successfully. Path: %s, Value: %s", podName, path, valueJSON))
	return nil
}

// DeleteAnnotation deletes an annotation specified in a path, in a pod in a given namespace
func (c *KubeClient) DeleteAnnotation(path, namespace, podName string) error {
	patchData := map[string]interface{}{
		"metadata": map[string]map[string]interface{}{
			"annotations": {
				path: nil,
			},
		},
	}

	payloadBytes, err := json.Marshal(patchData)
	if err != nil {
		logs.LogConf.Error("DeleteAnnotation: error while marshalling the annotation data", err)
		return err
	}

	_, updateErr := c.Client.CoreV1().Pods(namespace).Patch(podName, types.StrategicMergePatchType, payloadBytes)
	if updateErr != nil {
		fmt.Println("DeleteAnnotation: Error while deleting annotations:", updateErr)
		return updateErr
	}

	logs.LogConf.Info(fmt.Sprintf("DeleteAnnotation: Pod %s annotation %s deleted successfully.", podName, path))
	return nil
}

//DeletePod restart a pod
func (c *KubeClient) DeletePod(podName, namespace string) error {
	err := c.Client.CoreV1().Pods(namespace).Delete(podName, &metav1.DeleteOptions{})
	if err != nil {
		fmt.Println("error while delete the pod", podName, err)
		return err
	}

	return nil
}

//DeletePodWithGrace restarts a pod with a grace period
func (c *KubeClient) DeletePodWithGrace(podName, namespace string, GracePeriodSeconds *int64) error {
	err := c.Client.CoreV1().Pods(namespace).Delete(podName, &metav1.DeleteOptions{GracePeriodSeconds: GracePeriodSeconds})
	if err != nil {
		fmt.Println("error while delete the pod", podName, err)
		return err
	}

	return nil
}

//  patchUInt32Value specifies a patch operation for a uint32.
type patchUInt32Value struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value uint32 `json:"value"`
}

// GetReplicaCount gets the replicas in a deployment or a statefulset.
func (c *KubeClient) GetReplicaCount(namespace, deployment string) (int32, error) {
	var replicas int32
	deploymentDetails, depErr := c.GetDeployment(namespace, deployment)
	if depErr != nil {
		if strings.Contains(depErr.Error(), "not found") {
			fmt.Println("GetReplicaCount: Error while getting the deployment details.checking for statfulsets...")
			statefulsetDetails, sterr := c.GetStatefulSets(namespace, deployment)
			if sterr != nil {
				fmt.Println("GetReplicaCount: Error while getting the statefulset details(read replica num)", sterr)
				return -1, fmt.Errorf("Error while getting the statefulset details(read replica num) %s", sterr)
			}
			replicas = *statefulsetDetails.Spec.Replicas
		} else {
			fmt.Println("GetReplicaCount: Error while getting the deployment details(read replica num)", depErr)
			return -1, fmt.Errorf("Error while getting the deployment details(read replica num) %s", depErr)
		}
	}

	replicas = *deploymentDetails.Spec.Replicas
	return replicas, nil
}

// GetXGVelaNamespace get cluster role details
func (c *KubeClient) GetXGVelaNamespace() (string, error) {
	topoGwFqdn, err := c.GetTopoGwFqdn()
	if err != nil {
		return "", err
	}
	xgvelaNamespace := strings.Split(topoGwFqdn, ".")
	if len(xgvelaNamespace) >= 2 {
		return xgvelaNamespace[1], nil
	}
	return "", fmt.Errorf("XGVela Namespace could not be extracted from topogw.fqdn: %s", topoGwFqdn)
}

// GetTopoGwFqdn get topo-gw fqdn from annotations
func (c *KubeClient) GetTopoGwFqdn() (string, error) {
	//var top *TmaaSAnnotations

	topoGwFqdn := t.PodDetails.Annotations["topogw.fqdn"]
	if topoGwFqdn == "" {
		logs.LogConf.Error(
			"topogw.fqdn annotation is either not present or is blank.",
			"Falling back to check ClusterRole for topo gw fqdn",
		)
		// try clusterrole annotation if it is an older version
		var clusterRoleDetails *rbacv1.ClusterRole
		tmaasAnns, err := utility.GetAnnotations()
		if err != nil {
			return "", fmt.Errorf("error while getting TMaaS annotations. %s", err.Error())
		}
		for i := 0; i < t.ConfigDetails.RemoteSvcRetryCount; i++ {
			clusterRoleDetails, err = c.Client.RbacV1().ClusterRoles().Get(tmaasAnns.XGVelaID+"-xgvela", metav1.GetOptions{})
			if err == nil {
				return clusterRoleDetails.Annotations["topo-gw.fqdn"], nil
			}
			time.Sleep(1 * time.Second)
		}
		return "", fmt.Errorf("Error while getting topo-gw FQDN from Cluster Role. %s", err.Error())
	}

	return topoGwFqdn, nil
}

//RestartPod call k8s api to restart pod
func RestartPod(podID string) error {
	var err error
	logs.LogConf.Info(logs.LogService + "RestartPod: {[" + t.DeploymentID + "]-[" + t.Namespace + "]-[" + podID + "]-[HA_POD_RESTART]}")
	if KubeConnect == nil {
		KubeConnect, err = NewKubeConfig()
		if err != nil {
			logs.LogConf.Error("Error while connecting to k8s client while restarting the pod. Error:", err)
			return err
		}
	}
	namespace := t.Namespace
	if namespace == "" {
		namespace = os.Getenv("K8S_NAMESPACE")
	}
	//call k8s api
	err = KubeConnect.DeletePod(podID, namespace)
	if err != nil {
		logs.LogConf.Error(logs.LogService + "RestartPod: {[" + t.DeploymentID + "]-[" + t.Namespace + "]-[" + podID + "]-[HA_POD_RESTART-FAILED]}. Error: " + err.Error())
		return err
	}

	logs.LogConf.Info(logs.LogService + "RestartPod: {[" + t.DeploymentID + "]-[" + t.Namespace + "]-[" + podID + "]-[HA_POD_RESTART-SUCCESS]}")
	return nil
}
