// Copyright © by Jeff Foley 2017-2026. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.
// SPDX-License-Identifier: Apache-2.0

package support

import (
	"context"
	"errors"
	"net"
	"strconv"
	"strings"
	"time"

	jarm "github.com/caffix/jarm-go"
	et "github.com/owasp-amass/amass/v5/engine/types"
	amassnet "github.com/owasp-amass/amass/v5/internal/net"
	oam "github.com/owasp-amass/open-asset-model"
	oamdns "github.com/owasp-amass/open-asset-model/dns"
	"github.com/owasp-amass/open-asset-model/general"
	"github.com/owasp-amass/open-asset-model/network"
)

func JARMFingerprint(sess et.Session, target oam.Asset, portrel *general.PortRelation) (string, error) {
	var ipv6 bool
	var host string

	if fqdn, ok := target.(*oamdns.FQDN); ok {
		host = fqdn.Name
	} else if ip, ok := target.(*network.IPAddress); ok {
		ipv6 = ip.Address.Is6()
		host = ip.Address.String()
	} else {
		return "", errors.New("target must be a FQDN or IPAddress")
	}

	addr := host
	if ipv6 {
		addr = "[" + addr + "]"
	}
	addr += ":" + strconv.Itoa(portrel.PortNumber)

	var results []string
	for _, probe := range jarm.GetProbes(host, portrel.PortNumber) {
		sess.NetSem().Acquire()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		dial := amassnet.NewDialContext(5 * time.Second)
		c, err := dial(ctx, "tcp", addr)
		cancel()
		sess.NetSem().Release()
		if err != nil {
			return "", err
		}

		results = append(results, sendJARMProbe(c, probe))
		_ = c.Close()
	}

	hash := jarm.RawHashToFuzzyHash(strings.Join(results, ","))
	if hash == "00000000000000000000000000000000000000000000000000000000000000" {
		return "", errors.New("probes were not successful against the target")
	}
	return hash, nil
}

// sendJARMProbe sends one probe over the connection and returns the parsed server
// hello, or an empty string when the probe fails.
func sendJARMProbe(c net.Conn, probe jarm.JarmProbeOptions) string {
	_ = c.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if _, err := c.Write(jarm.BuildProbe(probe)); err != nil {
		return ""
	}

	_ = c.SetReadDeadline(time.Now().Add(5 * time.Second))
	buf := make([]byte, 1484)
	n, err := c.Read(buf)
	if err != nil || n == 0 {
		return ""
	}

	ans, err := jarm.ParseServerHello(buf[:n], probe)
	if err != nil {
		return ""
	}
	return ans
}
