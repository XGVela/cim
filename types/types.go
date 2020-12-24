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

package types

import (
	"time"

	"github.com/nats-io/go-nats"
	v1 "k8s.io/api/core/v1"
)

var (
	PodDetails                     *v1.Pod
	Namespace                      string
	ConfigmapName                  string
	PodID                          string
	MicroserviceName               string
	DeploymentID                   string
	InstanceID                     string
	ActivePods                     int32
	PassivePods                    int
	MsConfigVersion                string
	NfConfigVersion                string
	ConfigMapRevision              string
	AppPort                        string
	Userinputs                     *UserInput
	ModeOfDeployment               string
	AlertsReloader                 string
	NatsConnection                 *nats.Conn
	AppVersion                     string
	XGVelaRegistration             = make(chan bool)
	TMaasFqdn                      string
	ApigwRestFqdn                  string
	TopoEngineFqdn                 string
	ApigwRestURIPrefix             string
	XGVelaNamespace                string
	NfID                           string
	LOGTopicSubscribed             bool
	CIMRestPort                    int
	CIMNatsPort                    int
	ContainerID                    string
	ContainerName                  string
	AppShutDownCause               []string
	FlushAppLogs                   = make(chan bool)
	AppLogsFlushed                 bool
	AppShutDownCauseCimTermination = "AppShutDownCauseCimTermination"
)

const (
	ProtocolHTTP2 = "http2"
	ProtocolHTTP  = "http"
)

//UserInput details
type UserInput struct {
	Subject         []string
	QueueGroup      string
	MaxFileSizeMB   int
	MaxBackupFiles  int
	MaxAge          int
	BufSize         int
	PodID           string
	Namespace       string
	SubscriberCount int
	MsgCount        int
	FlushTimeout    time.Duration
}
