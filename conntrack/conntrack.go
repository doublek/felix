// Copyright (c) 2016-2017 Tigera, Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package conntrack

import (
	"syscall"

	log "github.com/Sirupsen/logrus"
	"github.com/vishvananda/netlink"

	"github.com/projectcalico/felix/ip"
	"github.com/projectcalico/felix/set"
)

type AnyIPMatchConntrackFilter struct {
	ips set.Set
}

func (f *AnyIPMatchConntrackFilter) MatchConntrackFlow(flow *netlink.ConntrackFlow) (match bool) {
	f.ips.Iter(func(item interface{}) error {
		ipB := item.(ip.Addr)
		ipAddr := ipB.AsNetIP()
		if ipAddr.Equal(flow.Forward.SrcIP) ||
			ipAddr.Equal(flow.Forward.DstIP) ||
			ipAddr.Equal(flow.Reverse.SrcIP) ||
			ipAddr.Equal(flow.Reverse.DstIP) {
			match = true
			return set.StopIteration
		}
		return nil
	})
	return
}

type Conntrack struct {
}

func New() *Conntrack {
	return &Conntrack{}
}

func (c Conntrack) RemoveConntrackFlows(ipVersion uint8, ipAddrs set.Set) {
	if ipAddrs.Len() == 0 {
		return
	}
	var family netlink.InetFamily
	switch ipVersion {
	case 4:
		family = syscall.AF_INET
	case 6:
		family = syscall.AF_INET6
	default:
		log.WithField("version", ipVersion).Panic("Unknown IP version")
	}
	log.Infof("Removing conntrack flows from table v%v for ips %v", ipVersion, ipAddrs)
	filter := &AnyIPMatchConntrackFilter{ips: ipAddrs}
	numFlows, err := netlink.ConntrackDeleteFilter(netlink.ConntrackTable, family, filter)
	if err != nil {
		log.Errorf("error when removing conntrack flows %v", err)
	} else {
		//TODO(doublek): Remove this log and else path.
		log.Infof("Successfully removed %v conntrack flows", numFlows)
	}
}
