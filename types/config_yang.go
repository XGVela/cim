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
	CimConfigObj       *ConfigObj
	NatsSubscribedSubj = make(map[string]bool)
)

type ConfigObj struct {
	CimConfig *CimConfig `json:"cimConfig"`
}
type CimConfig struct {
	Lmaas    *Lmaas    `json:"lmaas,omitempty"`
	App      *Config   `json:"cim_settings,omitempty"`
	Subjects *Subjects `json:"app_settings,omitempty"`
}

type Lmaas struct {
	MaxFileSizeInMB      int    `json:"max_file_size_in_mb"`
	MaxBackupFiles       int    `json:"max_backup_files"`
	MaxAge               int    `json:"max_age"`
	BufferSize           int    `json:"buffer_size"`
	FlushTimeout         int    `json:"flush_timeout"`
	LoggingMode          string `json:"logging_mode"`
	PreferLocalDeamonset bool   `json:"prefer_local_daemonset"`
}
