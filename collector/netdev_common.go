// Copyright 2015 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build !nonetdev
// +build linux freebsd openbsd dragonfly

package collector

import (
	"flag"
	"fmt"
	"regexp"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	netdevIgnoredDevices = flag.String(
		"collector.netdev.ignored-devices", "^$",
		"Regexp of net devices to ignore for netdev collector.")
)

type netDevCollector struct {
	subsystem             string
	ignoredDevicesPattern *regexp.Regexp
	metricDescs           map[string]*prometheus.Desc
	bytes_receive_hist    *prometheus.HistogramVec
	bytes_transmit_hist   *prometheus.HistogramVec

}

func init() {
	Factories["netdev"] = NewNetDevCollector
}

func NetworkBuckets(start, factor float64, count int) []float64{
	if count < 1 {
		panic("ExponentialBuckets needs a positive count")
	}
	if start <= 0 {
		panic("ExponentialBuckets needs a positive start value")
	}
	if factor <= 1 {
		panic("ExponentialBuckets needs a factor greater than 1")
	}
	buckets := make([]float64, count)
	for j:= 0; i < 10; i++ {
		if j == 0{
			placeholder := start
		}else{
			placeholder := start*j
		}
		for i := range buckets {
			buckets[i+(j*count)] = start
			placeholder /= factor
			start += placeholder
		}
	}
	return buckets
}

// NewNetDevCollector returns a new Collector exposing network device stats.
func NewNetDevCollector() (Collector, error) {
	pattern := regexp.MustCompile(*netdevIgnoredDevices)
	return &netDevCollector{
		subsystem:             "network",
		ignoredDevicesPattern: pattern,
		metricDescs:           map[string]*prometheus.Desc{},
		bytes_received_hist:   prometheus.NewHistogramVec(
					      prometheus.HistogramOpts{
					             Namespace: Namespace,
					             Subsystem: "network",
					             Name:      "bytes_received_hist",
					             Help:      "Histogram of network bytes received.",
					             Buckets:   NetworkBuckets(536870912, 2, 9),
					      },
				              []string{"device"},
                                       ),
		bytes_transmit_hist:   prometheus.NewHistogramVec(
					      prometheus.HistogramOpts{
					             Namespace: Namespace,
					             Subsystem: "network",
					             Name:      "bytes_received_hist",
					             Help:      "Histogram of network bytes transmitted.",
					             Buckets:   NetworkBuckets(536870912, 2, 9),
					      },
				              []string{"device"},
                                       ),

	}, nil
}

func (c *netDevCollector) Update(ch chan<- prometheus.Metric) (err error) {
	netDev, err := getNetDevStats(c.ignoredDevicesPattern)
	if err != nil {
		return fmt.Errorf("couldn't get netstats: %s", err)
	}
	for dev, devStats := range netDev {
		for key, value := range devStats {
			if key == "receive_bytes_hist"{
				c.bytes_receive_hist.WithLabelValues(dev).Observe(v)
			}else if key == "transmit_bytes_hist"{
				c.bytes_transmit_hist.WithLabelValues(dev).Observe(v)
			}else{
				desc, ok := c.metricDescs[key]
				if !ok {
					desc = prometheus.NewDesc(
						prometheus.BuildFQName(Namespace, c.subsystem, key),
						fmt.Sprintf("Network device statistic %s.", key),
						[]string{"device"},
						nil,
					)
					c.metricDescs[key] = desc
				}
				v, err := strconv.ParseFloat(value, 64)
				if err != nil {
					return fmt.Errorf("invalid value %s in netstats: %s", value, err)
				}
				ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, dev)
			}
		}
	}
	c.bytes_receive_hist.Collect(ch)
	c.bytes_transmit_hist.Collect(ch)
	return nil
}
