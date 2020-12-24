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

type EventDetails struct {
	EventName             string              `json:"eventName"`
	EventTime             int64               `json:"eventTime"`
	EventSourceType       string              `json:"eventSourceType,omitempty"`
	ManagedObject         []KeyValueObjects   `json:"managedObject"`
	AdditionalInfo        []KeyValueObjects   `json:"additionalInfo,omitempty"`
	ThresholdInfo         []KeyValueObjects   `json:"thresholdInfo,omitempty"`
	StateChangeDefinition []KeyValueObjects   `json:"stateChangeDefinition,omitempty"`
	MonitoredAttributes   []KeyValueObjects   `json:"monitoredAttributes,omitempty"`
	SourceHeaders         SourceHeaderDetails `json:"sourceHeaders"`
	ContainerId           string              `json:"containerId"`
	SourceName            string              `json:"sourceName,omitempty"`
	SourceID              string              `json:"sourceId,omitempty"`
}

type KeyValueObjects struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type SourceHeaderDetails struct {
	PodUUID       string `json:"pod_uuid,omitempty"`
	UHN           string `json:"uhn,omitempty"`
	CNFCUUID      string `json:"cnfc_uuid,omitempty"`
	PodID         string `json:"podId"`
	Microservice  string `json:"microservice"`
	Nf            string `json:"nf"`
	NfPrefix      string `json:"nfPrefix"`
	NfType        string `json:"nfType"`
	SvcVersion    string `json:"svcVersion"`
	VendorID      string `json:"vendorId,omitempty"`
	XGVelaID      string `json:"xgvelaId,omitempty"`
	NfClass       string `json:"nfClass,omitempty"`
	NfID          string `json:"nfId,omitempty"`
	NfServiceID   string `json:"nfServiceId,omitempty"`
	NfServiceType string `json:"nfServiceType,omitempty"`
}
