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

package collector

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"regexp"
	"strings"

	"github.com/prometheus/common/log"
)

var (
	procNetDevFieldSep = regexp.MustCompile("[ :] *")
	netBytesReceived = map[string]int{}
	netBytesTransmitted = map[string]int{}
)

func getNetDevStats(ignore *regexp.Regexp) (map[string]map[string]string, error) {
	file, err := os.Open(procFilePath("net/dev"))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return parseNetDevStats(file, ignore)
}

func parseNetDevStats(r io.Reader, ignore *regexp.Regexp) (map[string]map[string]string, error) {
	scanner := bufio.NewScanner(r)
	scanner.Scan() // skip first header
	scanner.Scan()
	parts := strings.Split(string(scanner.Text()), "|")
	if len(parts) != 3 { // interface + receive + transmit
		return nil, fmt.Errorf("invalid header line in net/dev: %s",
			scanner.Text())
	}

	header := strings.Fields(parts[1])
	netDev := map[string]map[string]string{}
	for scanner.Scan() {
		line := strings.TrimLeft(string(scanner.Text()), " ")
		parts := procNetDevFieldSep.Split(line, -1)
		if len(parts) != 2*len(header)+1 {
			return nil, fmt.Errorf("invalid line in net/dev: %s", scanner.Text())
		}

		dev := parts[0][:len(parts[0])]
		if ignore.MatchString(dev) {
			log.Debugf("Ignoring device: %s", dev)
			continue
		}
		netDev[dev] = map[string]string{}
		for i, v := range header {
			netDev[dev]["receive_"+v] = parts[i+1]
			netDev[dev]["transmit_"+v] = parts[i+1+len(header)]
		}
		if netBytesReceived[dev] == 0{
			placeholder, err := strconv.ParseFloat(parts[1])
			netBytesReceived[dev] = placeholder
			if err != nil {
				return nil, fmt.Errorf("invalid value for network bytes read: %s", err)
			}
			netDev[dev]["receive_bytes_hist"] = 0
		} else{
			currentV, err  := strconv.ParseFloat(parts[1], 64)
			if err != nil {
				return nil, fmt.Errorf("invalid value for network bytes read: %s", err)
			}
			previousV = netBytesReceived[dev]
			netDev[dev]["receive_bytes_hist"] = currentV - previousV
			netBytesReceived[dev] = currentV
		}
		if netBytesTransmitted[dev] == 0{
			placeholder, err := strconv.ParseFloat(parts[1])
			netBytesTransmitted[dev] = placeholder
			if err != nil {
				return nil, fmt.Errorf("invalid value for network bytes transmitted: %s", err)
			}
			netDev[dev]["transmit_bytes_hist"] = 0
		} else{
			currentV, err  := strconv.ParseFloat(parts[1+len(header)], 64)
			if err != nil {
				return nil, fmt.Errorf("invalid value for network bytes transmitted: %s", err)
			}
			previousV = netBytesTransmitted[dev]
			netDev[dev]["transmit_bytes_hist"] = currentV - previousV
			netBytesTransmitted[dev] = currentV
		}
	}
	return netDev, nil
}
