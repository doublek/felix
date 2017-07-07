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
	"bytes"
	"net"
	"os/exec"
	"strings"

	log "github.com/Sirupsen/logrus"

	"github.com/projectcalico/felix/set"
)

const numRetries = 3

type Conntrack struct {
	newCmd newCmd
}

func New() *Conntrack {
	return NewWithCmdShim(func(name string, arg ...string) CmdIface {
		return exec.Command(name, arg...)
	})
}

// NewWithCmdShim is a test constructor that allows for shimming exec.Command.
func NewWithCmdShim(newCmd newCmd) *Conntrack {
	return &Conntrack{
		newCmd: newCmd,
	}
}

type newCmd func(name string, arg ...string) CmdIface

type CmdIface interface {
	CombinedOutput() ([]byte, error)
}

func (c Conntrack) RemoveConntrackFlows(ipV4Addr set.Set, ipV6Addr set.Set) {
	buf := new(bytes.Buffer)
	ipV4Addr.Iter(func(item interface{}) error {
		ip := item.(net.IP)
		buf.WriteString(" --ipv4 ")
		buf.WriteString(ip.String())
		return nil
	})
	ipV6Addr.Iter(func(item interface{}) error {
		ip := item.(net.IP)
		buf.WriteString(" --ipv6 ")
		buf.WriteString(ip.String())
		return nil
	})
	log.Info("Removing conntrack flows")
	// Retry a few times because the conntrack command seems to fail at random.
	for retry := 0; retry <= numRetries; retry += 1 {
		cmd := c.newCmd("conntrack-delete", buf.String())
		output, err := cmd.CombinedOutput()
		if err == nil {
			log.Debug("Successfully removed conntrack flows.")
			break
		}
		if strings.Contains(string(output), "0 flow entries") {
			// Success, there were no flows.
			log.Debug("No IP wasn't in conntrack")
			break
		}
		if retry == numRetries {
			log.WithError(err).Error("Failed to remove conntrack flows after retries.")
		} else {
			log.WithError(err).Warn("Failed to remove conntrack flows, will retry...")
		}
	}
}
