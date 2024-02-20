// Copyright 2016 Markus Lindenberg
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
	"flag"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	//"github.com/prometheus/common/log"
	//"github.com/go-kit/log"

	"gopkg.in/mcuadros/go-syslog.v2"
)

const (
	namespace = "nginx_request"
)

func main() {

	var (
		listenAddress = flag.String("web.listen-address", ":9147", "Address to listen on for web interface and telemetry.")
		metricsPath   = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")
		syslogAddress = flag.String("nginx.syslog-address", ":9514", "Syslog listen address/socket for Nginx.")
		metricBuckets = flag.String("histogram.buckets", ".005,.01,.025,.05,.1,.25,.5,1,2.5,5,10", "Buckets for the Prometheus histogram.")
	)
	flag.Parse()

	// Parse the buckets
	var floatBuckets []float64
	for _, str := range strings.Split(*metricBuckets, ",") {
		bucket, err := strconv.ParseFloat(strings.TrimSpace(str), 64)
		if err != nil {
			log.Fatal(err)
		}
		floatBuckets = append(floatBuckets, bucket)
	}

	// Listen to signals
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGTERM, syscall.SIGINT)

	// Set up syslog server
	channel := make(syslog.LogPartsChannel, 20000)
	handler := syslog.NewChannelHandler(channel)
	server := syslog.NewServer()
	server.SetFormat(syslog.RFC3164)
	server.SetHandler(handler)

	var err error
	if strings.HasPrefix(*syslogAddress, "unix:") {
		err = server.ListenUnixgram(strings.TrimPrefix(*syslogAddress, "unix:"))
	} else {
		err = server.ListenUDP(*syslogAddress)
	}
	if err != nil {
		log.Fatal(err)

	}
	err = server.Boot()
	if err != nil {
		log.Fatal(err)
	}

	// Setup metrics
	syslogMessages := prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "exporter_syslog_messages",
		Help:      "Current total syslog messages received.",
	})
	err = prometheus.Register(syslogMessages)
	if err != nil {
		log.Fatal(err)
	}
	syslogParseFailures := prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "exporter_syslog_parse_failure",
		Help:      "Number of errors while parsing syslog messages.",
	})
	err = prometheus.Register(syslogParseFailures)
	if err != nil {
		log.Fatal(err)
	}
	var msgs int64
	go func() {
		for part := range channel {
			log.Printf("part:%s ", part)
			//part:map[client:172.21.244.36:59911 content:time:0.000 status=304  method="GET" upstream="-" facility:%!s(int=23) hostname:sre-file-ontest-1-c2zzm priority:%!s(int=190) severity:%!s(int=6) tag:nginx timestamp:2024-02-20 09:55:28 +0000 UTC tls_peer:]
			syslogMessages.Inc()
			msgs++
			tag, _ := part["tag"].(string)
			if tag != "nginx" {
				//log.Warn("Ignoring syslog message with wrong tag")
				log.Println("Ignoring syslog message with wrong tag")
				syslogParseFailures.Inc()
				continue
			}
			server, _ := part["hostname"].(string)
			if server == "" {
				log.Print("Hostname missing in syslog message")
				//log.Warn("Hostname missing in syslog message")
				syslogParseFailures.Inc()
				continue
			}

			content, _ := part["content"].(string)
			log.Printf("content:%s", content)
			//content:time:0.000 status=304  method="GET" upstream="-"
			if content == "" {
				log.Print("Ignoring empty syslog message")

				//log.Warn("Ignoring empty syslog message")
				syslogParseFailures.Inc()
				continue
			}

			metrics, labels, err := parseMessage(content)
			log.Printf("metrics:%s  labels:%s", metrics, labels)
			//metrics:[{time %!s(float64=0)}]  labels:&{[status method upstream] [304 GET -]}
			if err != nil {
				//log.Error(err)
				log.Fatal(err)
				continue
			}
			for _, metric := range metrics {
				var collector prometheus.Collector
				collector = prometheus.NewHistogramVec(prometheus.HistogramOpts{
					Namespace: namespace,
					Name:      metric.Name,
					Help:      fmt.Sprintf("Nginx request log value for %s", metric.Name),
					Buckets:   floatBuckets,
				}, labels.Names)
				if err := prometheus.Register(collector); err != nil {
					if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
						collector = are.ExistingCollector.(*prometheus.HistogramVec)
					} else {
						//log.Error(err)
						log.Print(err)
						continue
					}
				}
				collector.(*prometheus.HistogramVec).WithLabelValues(labels.Values...).Observe(metric.Value)
			}
		}
	}()

	// Setup HTTP server
	http.Handle(*metricsPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>Nginx Request Exporter</title></head>
             <body>
             <h1>Nginx Request Exporter</h1>
             <p><a href='` + *metricsPath + `'>Metrics</a></p>
             </body>
             </html>`))
	})

	go func() {
		//log.Infof("Starting Server: %s", *listenAddress)
		log.Printf("Starting Server: %s", *listenAddress)
		log.Fatal(http.ListenAndServe(*listenAddress, nil))
	}()

	s := <-sigchan
	//log.Infof("Received %v, terminating", s)
	log.Printf("Received %v, terminating", s)

	//log.Infof("Messages received: %d", msgs)
	log.Printf("Messages received: %d", msgs)

	err = server.Kill()
	if err != nil {
		//log.Error(err)
		log.Fatal(err)

	}
	os.Exit(0)
}
