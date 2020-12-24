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

package promutil

import (
	"fmt"
	"sync"
	"time"

	"github.com/go-mesh/openlogging"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	onceEnable sync.Once

	PromEnabled        bool
	PrometheusRegistry Registry
)

//PrometheusExporter is a prom exporter for cim
type PrometheusExporter struct {
	FlushInterval time.Duration
	lc            sync.RWMutex
	lg            sync.RWMutex
	ls            sync.RWMutex
	counters      map[string]*prometheus.CounterVec
	gauges        map[string]*prometheus.GaugeVec
	summaries     map[string]*prometheus.SummaryVec
	histograms    map[string]*prometheus.HistogramVec
}

//NewPrometheusExporter create a prometheus exporter
func NewPrometheusExporter(options Options) Registry {
	if PromEnabled {
		onceEnable.Do(func() {
			EnableRunTimeMetrics()
			openlogging.Info("go runtime metrics is exported")
		})

	}
	return &PrometheusExporter{
		FlushInterval: options.FlushInterval,
		lc:            sync.RWMutex{},
		lg:            sync.RWMutex{},
		ls:            sync.RWMutex{},
		summaries:     make(map[string]*prometheus.SummaryVec),
		counters:      make(map[string]*prometheus.CounterVec),
		gauges:        make(map[string]*prometheus.GaugeVec),
		histograms:    make(map[string]*prometheus.HistogramVec),
	}
}

// EnableRunTimeMetrics enable runtime metrics
func EnableRunTimeMetrics() {
	GetSystemPrometheusRegistry().MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	GetSystemPrometheusRegistry().MustRegister(prometheus.NewGoCollector())
}

//CreateGauge create collector
func (c *PrometheusExporter) CreateGauge(opts GaugeOpts) error {
	c.lg.RLock()
	_, ok := c.gauges[opts.Name]
	c.lg.RUnlock()
	if ok {
		return fmt.Errorf("metric [%s] is duplicated", opts.Name)
	}
	c.lg.Lock()
	defer c.lg.Unlock()
	gVec := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: opts.Name,
		Help: opts.Help,
	}, opts.Labels)
	c.gauges[opts.Name] = gVec
	GetSystemPrometheusRegistry().MustRegister(gVec)
	return nil
}

//GaugeSet set value
func (c *PrometheusExporter) GaugeSet(name string, val float64, labels map[string]string) error {
	c.lg.RLock()
	gVec, ok := c.gauges[name]
	c.lg.RUnlock()
	if !ok {
		return fmt.Errorf("metrics do not exists, create it first")
	}
	gVec.With(labels).Set(val)
	return nil
}

//CreateCounter create collector
func (c *PrometheusExporter) CreateCounter(opts CounterOpts) error {
	c.lc.RLock()
	_, ok := c.counters[opts.Name]
	c.lc.RUnlock()
	if ok {
		return fmt.Errorf("metric [%s] is duplicated", opts.Name)
	}
	c.lc.Lock()
	defer c.lc.Unlock()
	v := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: opts.Name,
		Help: opts.Help,
	}, opts.Labels)
	c.counters[opts.Name] = v
	GetSystemPrometheusRegistry().MustRegister(v)
	return nil
}

//CounterAdd increase value
func (c *PrometheusExporter) CounterAdd(name string, val float64, labels map[string]string) error {
	c.lc.RLock()
	v, ok := c.counters[name]
	c.lc.RUnlock()
	if !ok {
		return fmt.Errorf("metrics do not exists, create it first")
	}
	v.With(labels).Add(val)
	return nil
}

//CreateSummary create collector
func (c *PrometheusExporter) CreateSummary(opts SummaryOpts) error {
	c.ls.RLock()
	_, ok := c.summaries[opts.Name]
	c.ls.RUnlock()
	if ok {
		return fmt.Errorf("metric [%s] is duplicated", opts.Name)
	}
	c.ls.Lock()
	defer c.ls.Unlock()
	v := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name:       opts.Name,
		Help:       opts.Help,
		Objectives: opts.Objectives,
	}, opts.Labels)
	c.summaries[opts.Name] = v
	GetSystemPrometheusRegistry().MustRegister(v)
	return nil
}

//SummaryObserve set value
func (c *PrometheusExporter) SummaryObserve(name string, val float64, labels map[string]string) error {
	c.ls.RLock()
	v, ok := c.summaries[name]
	c.ls.RUnlock()
	if !ok {
		return fmt.Errorf("metrics do not exists, create it first")
	}
	v.With(labels).Observe(val)
	return nil
}

//CreateHistogram create collector
func (c *PrometheusExporter) CreateHistogram(opts HistogramOpts) error {
	c.ls.RLock()
	_, ok := c.histograms[opts.Name]
	c.ls.RUnlock()
	if ok {
		return fmt.Errorf("metric [%s] is duplicated", opts.Name)
	}
	c.ls.Lock()
	defer c.ls.Unlock()
	v := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    opts.Name,
		Help:    opts.Help,
		Buckets: opts.Buckets,
	}, opts.Labels)
	c.histograms[opts.Name] = v
	GetSystemPrometheusRegistry().MustRegister(v)
	return nil
}

//HistogramObserve set value
func (c *PrometheusExporter) HistogramObserve(name string, val float64, labels map[string]string) error {
	c.ls.RLock()
	v, ok := c.histograms[name]
	c.ls.RUnlock()
	if !ok {
		return fmt.Errorf("metrics do not exists, create it first")
	}
	v.With(labels).Observe(val)
	return nil
}

func init() {
	registries["prometheus"] = NewPrometheusExporter
}
