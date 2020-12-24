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
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var registries = make(map[string]NewRegistry)
var prometheusRegistry = prometheus.NewRegistry()

type NewRegistry func(opts Options) Registry

//Registry holds all of metrics collectors
//name is a unique ID for different type of metrics
type Registry interface {
	CreateGauge(opts GaugeOpts) error
	CreateCounter(opts CounterOpts) error
	CreateSummary(opts SummaryOpts) error
	CreateHistogram(opts HistogramOpts) error

	GaugeSet(name string, val float64, labels map[string]string) error
	CounterAdd(name string, val float64, labels map[string]string) error
	SummaryObserve(name string, val float64, Labels map[string]string) error
	HistogramObserve(name string, val float64, labels map[string]string) error
}

var defaultRegistry Registry

//CreateGauge init a new gauge type
func CreateGauge(opts GaugeOpts) error {
	return defaultRegistry.CreateGauge(opts)
}

//CreateCounter init a new counter type
func CreateCounter(opts CounterOpts) error {
	return defaultRegistry.CreateCounter(opts)
}

//CreateSummary init a new summary type
func CreateSummary(opts SummaryOpts) error {
	return defaultRegistry.CreateSummary(opts)
}

//CreateHistogram init a new summary type
func CreateHistogram(opts HistogramOpts) error {
	return defaultRegistry.CreateHistogram(opts)
}

//GaugeSet set a new value to a collector
func GaugeSet(name string, val float64, labels map[string]string) error {
	return defaultRegistry.GaugeSet(name, val, labels)
}

//CounterAdd increase value of a collector
func CounterAdd(name string, val float64, labels map[string]string) error {
	return defaultRegistry.CounterAdd(name, val, labels)
}

//SummaryObserve gives a value to summary collector
func SummaryObserve(name string, val float64, labels map[string]string) error {
	return defaultRegistry.SummaryObserve(name, val, labels)
}

//HistogramObserve gives a value to histogram collector
func HistogramObserve(name string, val float64, labels map[string]string) error {
	return defaultRegistry.HistogramObserve(name, val, labels)
}

//CounterOpts is options to create a counter options
type CounterOpts struct {
	//fmt.Println("inside counterOpts struct")
	Name   string
	Help   string
	Labels []string
}

//GaugeOpts is options to create a gauge collector
type GaugeOpts struct {
	//fmt.Println("inside counterOpts struct")
	Name   string
	Help   string
	Labels []string
}

//SummaryOpts is options to create summary collector
type SummaryOpts struct {
	Name       string
	Help       string
	Labels     []string
	Objectives map[float64]float64
}

//HistogramOpts is options to create histogram collector
type HistogramOpts struct {
	Name    string
	Help    string
	Labels  []string
	Buckets []float64
}

//Options control config
type Options struct {
	FlushInterval time.Duration
}

//InstallPlugin install metrics registry
func InstallPlugin(name string, f NewRegistry) {
	registries[name] = f
}

//Init load the metrics plugin and initialize it
func Init() error {
	var name string
	name = "prometheus"
	f, ok := registries[name]
	if !ok {
		return fmt.Errorf("can not init metrics registry [%s]", name)
	}

	defaultRegistry = f(Options{
		FlushInterval: 10 * time.Second,
	})

	return nil
}

//GetSystemPrometheusRegistry return prometheus registry which cim use
func GetSystemPrometheusRegistry() *prometheus.Registry {
	return prometheusRegistry
}
