// Copyright © by Jeff Foley 2017-2026. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.
// SPDX-License-Identifier: Apache-2.0

package scrape

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIPVerseReserveIsAtomic(t *testing.T) {
	v := NewIPVerse().(*ipverse)
	const asn = 64496 // reserved for documentation (RFC 5398)

	var wg sync.WaitGroup
	var claimed atomic.Int32
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if v.reserve(asn) {
				claimed.Add(1)
			}
		}()
	}
	wg.Wait()
	assert.Equal(t, int32(1), claimed.Load(), "exactly one lookup may claim an ASN")

	v.release(asn)
	assert.True(t, v.reserve(asn), "a released ASN can be claimed again")
}
