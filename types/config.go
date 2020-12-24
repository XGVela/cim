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

var (
	ConfigDetails        *Config
	CommonInfraConfigObj *CommonInfraConf
	InTopic              string
	OutTopic             string
)

type Config struct {
	// KafkaEnabled       bool     `json:"kafka_enabled"`
	EnableKpi           bool   `json:"enable_kpi,omitempty"`
	KafkaClientID       string `json:"kafka_client_id,omitempty"`
	AppPort             string `json:"app_port,omitempty"`
	PrometheusEnabled   bool   `json:"prometheus_enabled,omitempty"`
	InsecureSkipVerify  bool   `json:"insecure_skip_verify,omitempty"` // Http2 enforces to use certs, if it's true client no need to send key.
	AllowHttp           bool   `json:"allow_http,omitempty"`           // if url contains https, then also handled
	KeyFilePath         string `json:"key_file_path,omitempty"`        // if insecureskipverify false, key file path
	LogLevel            string `json:"log_level,omitempty"`
	CimFileLog          bool   `json:"cim_file_log,omitempty"`
	NumGarpCount        int    `json:"num_garp_count,omitempty"`
	RemoteSvcRetryCount int    `json:"remote_svc_retry_count,omitempty"`
	TTLTimeout          int    `json:"ttl_timeout,omitempty"`
}
type Subjects struct {
	AppPort      int  `json:"app_port"`
	Http2Enabled bool `json:"http2_enabled"` // Http2 supported if true
}
type CommonInfraConf struct {
	EtcdURL           string   `json:"etcd_url"`
	KafkaBrokers      []string `json:"kafka_brokers"`
	LoggingSvcUrl     string   `json:"logging_svc_url"`
	LoggingSvcTcpPort string   `json:"logging_svc_tcp_port"`
	EnableRetx        string   `json:"enable_retrx"`
	APIGwFqdn         string   `json:"apigw_rest_fqdn"`
	TopoEngineFqdn    string   `json:"topoengine_fqdn"`
	LogFormat         string   `json:"log_format"`
}
