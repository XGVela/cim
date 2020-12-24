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

package clockalignedmetrics

import (
	httpUtility "cim/http_utility"
	"cim/logs"
	"cim/types"
	"encoding/json"
	"fmt"

	"io/ioutil"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

const (
	closed = "CLOSED"
	open   = "OPEN"
)

type clockAlignedConfig struct {
	ScrapeEnabled string   `json:"scrape"`
	Path          string   `json:"path"`
	Ports         []string `json:"ports"`
	Scheme        string   `json:"scheme"`
	MeasInterval  string   `json:"measurementInterval"`
	MaxRetention  string   `json:"maxRetention"`
}

var (
	mc      *MetricsCollector
	success float64
	once    sync.Once
	//maxRetention = 10 // no of bucket
	config clockAlignedConfig
)

// MetricsCollector implements the prometheus.Collector interface.
type MetricsCollector struct {
	Buckets map[int]*Bucket
}

type Bucket struct {
	BucketID      int
	Status        string
	CaFetchedData []*dto.MetricFamily
	TimeStamp     time.Time
}

type handler struct {
	// exporterMetricsRegistry is a separate registry for the metrics.
	exporterMetricsRegistry *prometheus.Registry
}

func newMetricsCollector() *MetricsCollector {
	maxNumBuckets, _ := strconv.Atoi(config.MaxRetention)
	buckets := make(map[int]*Bucket)
	for i := 1; i <= maxNumBuckets; i++ {
		b := &Bucket{
			BucketID:      i,
			Status:        open,
			CaFetchedData: []*dto.MetricFamily{},
		}
		buckets[i] = b
	}
	return &MetricsCollector{Buckets: buckets}
}

//RunAppMetricCollection start collecting metrics from app
func RunAppMetricCollection() {
	//logs.LogConf.Info("cim started at:", time.Now())
	currentTime := time.Now().Unix()
	logs.LogConf.Info("measurement interval is:", config.MeasInterval)
	measInterval, err := time.ParseDuration(config.MeasInterval)
	if err != nil {
		logs.LogConf.Error("error while getting the interval", err)
		return
	}

	nextScrapeTime := int(measInterval.Seconds()) - int((float64(currentTime)+measInterval.Seconds()))%int(measInterval.Seconds())
	//nextScrapeTime := int(measInterval.Seconds()) - int(currentTime)%int(measInterval.Seconds())

	logs.LogConf.Info("nextscrapetime within:", time.Duration(nextScrapeTime)*time.Second)

	//start ticker
	ticker := time.NewTicker(time.Duration(nextScrapeTime) * time.Second)

	go func() {
		for {
			select {
			case <-ticker.C:
				//scrape metrics from app
				scrapetime := time.Now().Round(5 * time.Second)
				once.Do(func() {
					ticker = time.NewTicker(measInterval)
				})
				//logs.LogConf.Info("scraping time :", scrapetime)
				for bucketID, details := range mc.Buckets {

					if details.Status == open {
						collectMetrics(bucketID, scrapetime)
						break
					}
				}

			}
		}
	}()
}

func collectMetrics(id int, scrapeTime time.Time) {

	var (
		parser expfmt.TextParser
		result []*dto.MetricFamily
	)

	//scrape application metrics
	for _, port := range config.Ports {
		appResp, err := httpUtility.HttpGet(config.Scheme+"://localhost:"+port+config.Path, "application/json", nil)
		if err != nil {
			logs.LogConf.Error("error while getting the metrics from app", err, time.Now())
		}

		if appResp != nil && appResp.Body != nil {
			appBody, err := ioutil.ReadAll(appResp.Body)
			if err != nil {
				logs.LogConf.Error("error while reading the app response body", err)
			}

			appResp.Body.Close()
			appMetrics, err := parser.TextToMetricFamilies(strings.NewReader(string(appBody)))
			if err != nil {
				logs.LogConf.Error("error while parsing app text to metric family", err)
			}

			for _, mf := range appMetrics {
				for _, metric := range mf.Metric {
					var sourcePort = new(dto.LabelPair)
					name := "source_port"
					sport := port
					sourcePort.Name = &name
					sourcePort.Value = &sport
					metric.Label = append(metric.Label, sourcePort)
					result = append(result, mf)
				}
			}

		}
	}

	//scrape cim metrics
	cimResp, err := httpUtility.HttpGet(fmt.Sprintf("http://localhost:%d/metrics", types.CIMRestPort), "application/json", nil)
	if err != nil {
		logs.LogConf.Error("error while getting the metrics from cim", err, time.Now())
	}

	if cimResp != nil && cimResp.Body != nil {

		cimBody, err := ioutil.ReadAll(cimResp.Body)
		if err != nil {
			logs.LogConf.Error("error while reading the cim response body", err)
		}

		cimResp.Body.Close()

		cimMetrics, err := parser.TextToMetricFamilies(strings.NewReader(string(cimBody)))
		if err != nil {
			logs.LogConf.Error("error while parsing cim text to metric family", err)
		}

		for _, mf := range cimMetrics {
			for _, metric := range mf.Metric {
				var sourcePort = new(dto.LabelPair)
				name := "source_port"
				val := strconv.Itoa(types.CIMRestPort)
				sourcePort.Name = &name
				sourcePort.Value = &val
				metric.Label = append(metric.Label, sourcePort)
				result = append(result, mf)
			}
		}
	}

	if result != nil {
		mc.Buckets[id].CaFetchedData = result
		mc.Buckets[id].Status = closed
		mc.Buckets[id].TimeStamp = scrapeTime
	}
}

//IsClockAlignedMetricsEnabled check if clock aligned metrics is enabled
func IsClockAlignedMetricsEnabled() bool {
	mmaasAnnotations := types.PodDetails.Annotations["xgvela.com/mmaas"]
	logs.LogConf.Info("mmaas annotation details are:", mmaasAnnotations)
	if mmaasAnnotations == "" {
		logs.LogConf.Info("clock aligned metrics exposition is disabled")
		return false
	}

	err := json.Unmarshal([]byte(mmaasAnnotations), &config)
	if err != nil {
		logs.LogConf.Error("error while getting mmaas annotations", err)
		return false
	}

	if config.ScrapeEnabled == "true" {
		return true
	}

	return false
}
