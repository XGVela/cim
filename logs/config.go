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

package logs

import (
	"log"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type KubeClient struct {
	Client *kubernetes.Clientset
}

var kubeconfig = ""

//NewKubeConfig get k8s client connection
func NewKubeConfig() (*KubeClient, error) {
	config, err := getConfig(kubeconfig)
	if err != nil {
		log.Printf("Failed to load client config: %v", err)
		return nil, err
	}
	// build the Kubernetes client
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Printf("Failed to create kubernetes client: %v", err)
		return nil, err
	}

	k8sconfig := &KubeClient{
		Client: client,
	}

	return k8sconfig, nil

}

func getConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}
