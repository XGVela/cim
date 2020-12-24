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

package metrics

import (
	"cim/logs"
	"cim/promutil"
	"log"
)

//LogMetricInit function to register log metrics
func ConfigMetricInit() {
	var err error

	err = promutil.CreateCounter(promutil.CounterOpts{
		Name:   "cim_config_push_attempt_total",
		Help:   "CIM-Metric for total number of Config push attempt",
		Labels: []string{"pod"},
	})

	if err != nil {
		logs.LogConf.Error("error while creating counter metric for total config push attempt", err)
		return
	}

	err = promutil.CreateCounter(promutil.CounterOpts{
		Name:   "cim_config_push_failure_total",
		Help:   "CIM-Metric for total number of Config attempt failed. ",
		Labels: []string{"pod"},
	})
	if err != nil {
		log.Println("error while creating counter metric for total config push failure", err)
		return
	}

	err = promutil.CreateCounter(promutil.CounterOpts{
		Name:   "cim_config_push_success_total",
		Help:   "CIM-Metric for total number of Config push success",
		Labels: []string{"pod"},
	})
	if err != nil {
		logs.LogConf.Error("error while creating counter metric for total config push success", err)
		return
	}
	err = promutil.CreateCounter(promutil.CounterOpts{
		Name:   "cim_config_resp_received_total",
		Help:   "CIM-Metric for total number of Config response received. ",
		Labels: []string{"pod"},
	})

	if err != nil {
		logs.LogConf.Error("error while creating counter metric for total config response recieved", err)
		return
	}

	err = promutil.CreateCounter(promutil.CounterOpts{
		Name:   "cim_commit_config_push_failure_total",
		Help:   "CIM-Metric for total number of Commit config  push attempt failed.",
		Labels: []string{"pod"},
	})
	if err != nil {
		log.Println("error while creating counter metric for total commit config push failure", err)
		return
	}
	err = promutil.CreateCounter(promutil.CounterOpts{
		Name:   "cim_config_resp_success_total",
		Help:   "CIM-Metric for total number of Config resp success .",
		Labels: []string{"pod"},
	})

	if err != nil {
		logs.LogConf.Error("error while creating counter metric for total config response success", err)
		return
	}

	err = promutil.CreateCounter(promutil.CounterOpts{
		Name:   "cim_config_resp_failure_total",
		Help:   "CIM-Metric for total number of Config resp failure.",
		Labels: []string{"pod"},
	})
	if err != nil {
		log.Println("error while creating counter metric for total config response failure", err)
		return
	}
	err = promutil.CreateCounter(promutil.CounterOpts{
		Name:   "cim_config_resp_unused_total",
		Help:   "CIM-Metric for total number of Config resp unused.",
		Labels: []string{"pod"},
	})
	if err != nil {
		log.Println("error while creating counter metric for total config response unused", err)
		return
	}

}
