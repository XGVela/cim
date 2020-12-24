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
	"cim/logs"
	//"cim/notification"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/version"
	"net/http"
	"strconv"
	"time"
)

var scrapeSuccessDesc = prometheus.NewDesc(
	prometheus.BuildFQName("metric", "scrape", "collector_success"),
	"metric_exporter: Whether a collector succeeded.",
	[]string{"collector"},
	nil,
)

//Init initialize clock aligned metrics handler
func Init() http.Handler {

	mc = newMetricsCollector()

	r := prometheus.NewRegistry()
	r.MustRegister(version.NewCollector("clock_aligned_metric_exporter"))
	if err := r.Register(mc); err != nil {
		//TODO: handle event notify
		//notify_util.RaiseEvent("ClockAlignedMetricFailed", []string{"Description", "STATUS"}, []string{"Registration Failure", "Failure"}, []string{"Error"}, []string{err.Error()})
		logs.LogConf.Exception("ClockAlignedMetricFailed: couldn't register metric collector:", err)
	}

	handler := promhttp.HandlerFor(
		r,
		promhttp.HandlerOpts{
			ErrorHandling: promhttp.ContinueOnError,
		},
	)

	return handler
}

// Describe implements the prometheus.Collector interface.
func (b *MetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- scrapeSuccessDesc
}

//Collect implements the prometheus.Collector interface.
func (b *MetricsCollector) Collect(ch chan<- prometheus.Metric) {
	var mTime time.Time
	for bucketID, data := range mc.Buckets {
		if data.Status == closed {
			for _, mf := range data.CaFetchedData {
				allLabelNames := map[string]struct{}{}
				for _, metric := range mf.Metric {
					labels := metric.GetLabel()
					for _, label := range labels {
						if _, ok := allLabelNames[label.GetName()]; !ok {
							allLabelNames[label.GetName()] = struct{}{}
						}
					}
				}

				for _, metric := range mf.Metric {
					mTime = data.TimeStamp
					tm := metric.GetTimestampMs()
					if tm != 0 {
						mTime = time.Unix(tm, 0)
					}
					execute(mf, metric, ch, allLabelNames, mTime, bucketID)
				}

			}

			mc.Buckets[bucketID].Status = open
			success = 1
			ch <- prometheus.MustNewConstMetric(scrapeSuccessDesc, prometheus.CounterValue, success, "metric")

		}
	}

}

func execute(metricFamily *dto.MetricFamily, metric *dto.Metric, ch chan<- prometheus.Metric, allLabelNames map[string]struct{}, t time.Time, id int) {
	var valType prometheus.ValueType
	var val float64

	labels := metric.GetLabel()
	var names []string
	var values []string
	for _, label := range labels {
		names = append(names, label.GetName())
		values = append(values, label.GetValue())
	}

	names = append(names, "aligned_ts")
	values = append(values, strconv.FormatInt(t.Unix(), 10))
	for k := range allLabelNames {
		present := false
		for _, name := range names {
			if k == name {
				present = true
				break
			}
		}
		if !present {
			names = append(names, k)
			values = append(values, "")
		}
	}

	metricType := metricFamily.GetType()
	switch metricType {
	case dto.MetricType_COUNTER:
		valType = prometheus.CounterValue
		val = metric.Counter.GetValue()

	case dto.MetricType_GAUGE:
		valType = prometheus.GaugeValue
		val = metric.Gauge.GetValue()

	case dto.MetricType_UNTYPED:
		valType = prometheus.UntypedValue
		val = metric.Untyped.GetValue()

	case dto.MetricType_SUMMARY:
		quantiles := map[float64]float64{}
		for _, q := range metric.Summary.Quantile {
			quantiles[q.GetQuantile()] = q.GetValue()
		}
		ch <- prometheus.NewMetricWithTimestamp(
			t,
			prometheus.MustNewConstSummary(
				prometheus.NewDesc(
					*metricFamily.Name,
					metricFamily.GetHelp(),
					names, nil,
				),
				metric.Summary.GetSampleCount(),
				metric.Summary.GetSampleSum(),
				quantiles, values...,
			))
	case dto.MetricType_HISTOGRAM:
		hisBuckets := map[float64]uint64{}
		for _, b := range metric.Histogram.Bucket {
			hisBuckets[b.GetUpperBound()] = b.GetCumulativeCount()
		}
		ch <- prometheus.NewMetricWithTimestamp(
			t,
			prometheus.MustNewConstHistogram(
				prometheus.NewDesc(
					*metricFamily.Name,
					metricFamily.GetHelp(),
					names, nil,
				),
				metric.Histogram.GetSampleCount(),
				metric.Histogram.GetSampleSum(),
				hisBuckets, values...,
			))
	default:
		logs.LogConf.Info("unknown metric type", metricType)
	}

	if metricType == dto.MetricType_GAUGE || metricType == dto.MetricType_COUNTER || metricType == dto.MetricType_UNTYPED {
		ch <- prometheus.NewMetricWithTimestamp(
			t,
			prometheus.MustNewConstMetric(
				prometheus.NewDesc(
					*metricFamily.Name,
					metricFamily.GetHelp(),
					names, nil,
				),
				valType, val, values...,
			))
	}
}
