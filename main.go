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

package main

import (
	"cim/agents"
	common_infra "cim/common-infra"
	"cim/config"
	"cim/connection"
	httpUtility "cim/http_utility"
	"cim/kubernetes"
	"cim/logs"
	"cim/metrics"
	clockalignedmetrics "cim/metrics/clock_aligned_metrics"
	notify_util "cim/notification"
	"cim/promutil"
	"cim/register"
	"cim/subscriptions"
	"cim/types"
	"cim/utility"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"runtime/debug"
	"time"

	"github.com/gorilla/mux"
	"github.com/nats-io/go-nats"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/soheilhy/cmux"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"golang.org/x/sync/errgroup"
)

func usage() {
	logs.LogConf.Info("Usage: nats-log-sub [-s server] <subject> <queue>\n")
	flag.PrintDefaults()
}

func init() {
	types.CurrentCimState = types.CimInitializing
}

func serveHTTP(l net.Listener, mux http.Handler) error {

	h2s := &http2.Server{}

	server := &http.Server{
		Handler: h2c.NewHandler(mux, h2s),
	}

	if err := server.Serve(l); err != cmux.ErrListenerClosed {
		return err
	}
	return nil
}

func serveHTTPS(l net.Listener, mux http.Handler) error {

	config := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{http2.ClientPreface},
	}
	// setup Server
	s := &http.Server{
		Handler:   mux,
		TLSConfig: config,
	}

	if err := s.ServeTLS(l, "server.crt", "server.key"); err != cmux.ErrListenerClosed {
		return err
	}
	return nil
}

func handlePanic() {
	var err error
	logs.LogConf.Error(string(debug.Stack()))

	podID := os.Getenv("K8S_POD_ID")
	if types.CurrentCimState >= types.CimStateRegistered {
		err = kubernetes.RestartPod(podID)
		if err != nil {
			logs.LogConf.Exception("Error while restarting the pod during panic.", err)
		}

		logs.LogConf.Info("Restarted pod:", podID, "because of panic")
	}
}

func main() {

	defer func() {
		handlePanic()
	}()

	//get all the env variables set
	utility.GetAllEnvironmentVariablesSet()

	var serveropts = server.Options{
		Host:           "127.0.0.1",
		Port:           types.CIMNatsPort,
		NoLog:          true,
		NoSigs:         true,
		MaxControlLine: 2048,
	}

	s, err := server.NewServer(&serveropts)
	if err != nil {
		fmt.Println("error while nats server start up", err.Error())
		return
	}
	go s.Start()
	connectionEstablished := s.ReadyForConnections(30 * time.Second)
	fmt.Println("NATs service started:", connectionEstablished, ", on port:", types.CIMNatsPort)

	opts := []nats.Option{nats.Name("NATS Log Subscriber")}
	opts = connection.SetupConnOptions(opts)
	var urls = flag.String("s", fmt.Sprintf("nats://localhost:%d", types.CIMNatsPort), "The nats server URLs (separated by comma)")
	types.NatsConnection, err = connection.NewConnection(*urls, opts...)
	if err != nil {
		log.Println("nats server is not running. please check", err.Error())
		os.Exit(1)
	}

	//read annotation from pod template
	kubernetes.KubeConnect, err = kubernetes.NewKubeConfig()
	if err != nil {
		logs.LogConf.Exception(logs.LogService+"[KubeConnect]: failed.Error while connecting to kubernetes client", err)
	}

	RemoteSvcRetryCount := 60
	for i := 0; i < RemoteSvcRetryCount; i++ {
		types.PodDetails, err = kubernetes.KubeConnect.GetPod(types.Namespace, types.PodID)
		if err != nil {
			logs.LogConf.Warning(logs.LogService + "[GET-POD]: {[" + types.PodID + "]} failed.Error while getting the pod details(read annotations)." + err.Error() + ". Retrying.....")
			time.Sleep(1 * time.Second)
			continue
		}
		break
	}

	if err != nil {
		logs.LogConf.Exception(logs.LogService + "[GET-POD]:  Error!!! Max attempts exhausted, unable to fetch k8s pod details. Exiting...")
	}

	tmaasAnna, e := utility.GetAnnotations()
	if e != nil {
		logs.LogConf.Exception("Error While Reading Tmaas Annotaion", e)
	}

	types.AppVersion = types.PodDetails.Annotations["svcVersion"]
	if types.AppVersion == "" {
		types.AppVersion = "v0"
	}
	logs.LogConf.Info("app version is ", types.AppVersion)

	types.ConfigmapName = tmaasAnna.NfServiceType + "-" + "cim" + "-" + types.AppVersion + "-" + "mgmt-cfg"
	types.NfID = tmaasAnna.NfID
	types.MicroserviceName = types.PodDetails.Labels["microSvcName"]
	config.LoadDefaultCimConfig()
	logs.InitializeLogInfo(types.CimConfigObj.CimConfig.App.LogLevel, types.CimConfigObj.CimConfig.App.CimFileLog)

	httpUtility.InitializeHttpClient()
	common_infra.ReadCommonInfraConf()

	agents.LogFormatJson = types.CommonInfraConfigObj.LogFormat != "text"
	if types.CimConfigObj.CimConfig.App.EnableKpi {
		promutil.PromEnabled = true
		//init metrics registry by prometheus name and setting the options in default registry
		if err := promutil.Init(); err != nil {
			log.Println("error while registring the prometheus", err)
			return
		}

		fmt.Println("conf", promutil.PrometheusRegistry)
		//creating counters and setting the specified metrics in default registry
		metrics.LogMetricInit()
		metrics.ConfigMetricInit()
	}

	//Try kafka dial
	connection.ConnKafka()
	//establish a connection with etcd
	_, etcd_err := connection.EtcdConnect()
	if etcd_err != nil {
		// Raise event in case of etcd failure
		notify_util.RaiseEvent("CimConnectEtcdFailure", []string{"Description", "STATUS"}, []string{"Connect Etcd Failure", "Failure"}, []string{"Error"}, []string{err.Error()})
		logs.LogConf.Exception("Unable to connect etcd Terminanting CIM")
	}

	switch types.CimConfigObj.CimConfig.Lmaas.LoggingMode {
	case "TCP":
		logs.LogConf.Info("Logging Method is TCP")
		agents.InitialiseTCPLoggingParams()
		go agents.CreateConnection()
	case "KAFKA":
		logs.LogConf.Info("Logging Method is KAFKA")
		agents.KafkaLogConn, _ = agents.IntialiseKafkaProducer()
	default:
		logs.LogConf.Info("Logging Method is", types.CimConfigObj.CimConfig.Lmaas.LoggingMode)
	}

	//define a router object to handle apis
	r := mux.NewRouter()

	if clockalignedmetrics.IsClockAlignedMetricsEnabled() {
		caMetricHandler := clockalignedmetrics.Init()
		logs.LogConf.Info("Clock aligned Metrics is enabled")

		//clockaligned metric handler
		r.Handle("/metrics/aligned", caMetricHandler)
		clockalignedmetrics.RunAppMetricCollection()
	} else {
		logs.LogConf.Info("Clock aligned Metrics is disabled")
	}

	os.Setenv("ETCDCTL_API", "3")

	//cim metric handler
	r.Handle("/metrics", promhttp.HandlerFor(promutil.GetSystemPrometheusRegistry(), promhttp.HandlerOpts{}))
	r.HandleFunc("/api/v1/_operations/xgvela/register", register.XGVelaRegistration).Methods(http.MethodPost)

	types.Userinputs, err = utility.GetUserInputs()
	notify_util.InputDetails = types.Userinputs

	if err != nil {
		logs.LogConf.Exception(err.Error())
	}
	// Connect Options.
	if types.NatsConnection != nil {
		defer types.NatsConnection.Close()
	}

	logs.LogConf.Info("Started watching etcd key: ", "change-set/"+types.Namespace)

	go agents.WatchConfigChange(connection.ConnectEtcd, "change-set/"+types.Namespace+"/"+types.MicroserviceName, types.ConfigDetails.AppPort)
	go agents.WatchDayOneConfig()
	notify_util.InitNotificationFramework()
	go func() {
		select {
		case <-types.XGVelaRegistration:
			r.HandleFunc("/api/v1/_operations/event/subscriptions", subscriptions.SubscribeEvents).Methods(http.MethodPost)
			r.HandleFunc("/api/v1/_operations/event/subscriptions", subscriptions.GetSubscribedEvents).Methods(http.MethodGet)
			r.HandleFunc("/api/v1/_operations/event/subscriptions/{subscriptionId}", subscriptions.UnsubscribeEvents).Methods(http.MethodDelete)
			r.HandleFunc("/api/v1/_operations/event/subscriptions/{subscriptionId}", subscriptions.UpdateSubscribedEvents).Methods(http.MethodPut)
			if types.CimConfigObj.CimConfig.App.EnableKpi {
				metrics.NatsMetricInit()
			}
		}
	}()
	go agents.HandleCIMTermination()

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", types.CIMRestPort))
	if err != nil {
		logs.LogConf.Exception(err.Error())
	}

	tcpm := cmux.New(l)

	// Declare the matchers for different services required.
	httpl := tcpm.Match(cmux.HTTP2(), cmux.HTTP1()) // it will match http and h2c
	httpsl := tcpm.Match(cmux.TLS())                // it will match tls irrespective of http version

	g := new(errgroup.Group)
	g.Go(func() error { return serveHTTP(httpl, r) })
	g.Go(func() error { return serveHTTPS(httpsl, r) })
	g.Go(func() error { return tcpm.Serve() })

	logs.LogConf.Info(fmt.Sprintf("Listening on %d for HTTP/HTTP2 requests", types.CIMRestPort))

	logs.LogConf.Exception("Server error, if any:", g.Wait())
}
