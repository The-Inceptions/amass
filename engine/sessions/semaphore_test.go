// Copyright © by Jeff Foley 2017-2026. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.
// SPDX-License-Identifier: Apache-2.0

package sessions

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// acquireAsync calls Acquire in a new goroutine and returns a channel that is closed once it returns.
func acquireAsync(ss *sessSemaphore) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		ss.Acquire()
		close(done)
	}()
	return done
}

func returnsWithin(ch <-chan struct{}, d time.Duration) bool {
	select {
	case <-ch:
		return true
	case <-time.After(d):
		return false
	}
}

func TestSessSemaphoreReleaseUnblocksAcquire(t *testing.T) {
	ss := NewSessSemaphore(2)
	ss.Acquire()
	ss.Acquire()

	done := acquireAsync(ss)
	require.False(t, returnsWithin(done, 50*time.Millisecond), "Acquire must block while every slot is held")

	released := make(chan struct{})
	go func() {
		ss.Release()
		close(released)
	}()
	require.True(t, returnsWithin(released, time.Second), "Release must not block behind a waiting Acquire")
	require.True(t, returnsWithin(done, time.Second), "the waiting Acquire must proceed after a Release")
}

func TestSessSemaphoreShrinkLimit(t *testing.T) {
	ss := NewSessSemaphore(3)
	for range 3 {
		ss.Acquire()
	}

	ss.SetLimit(1)
	ss.Release()

	// two holders remain, which is still above the new limit of one
	done := acquireAsync(ss)
	require.False(t, returnsWithin(done, 50*time.Millisecond))

	ss.Release()
	require.False(t, returnsWithin(done, 50*time.Millisecond))

	ss.Release()
	require.True(t, returnsWithin(done, time.Second))
}

func TestSessSemaphoreGrowLimit(t *testing.T) {
	ss := NewSessSemaphore(1)
	ss.Acquire()

	done := acquireAsync(ss)
	require.False(t, returnsWithin(done, 50*time.Millisecond))

	ss.SetLimit(2)
	require.True(t, returnsWithin(done, time.Second))
}

func TestSessSemaphoreOverRelease(t *testing.T) {
	ss := NewSessSemaphore(1)
	ss.Release()
	ss.Release()

	ss.Acquire()
	done := acquireAsync(ss)
	assert.False(t, returnsWithin(done, 50*time.Millisecond), "extra releases must not raise the limit")

	ss.Release()
	assert.True(t, returnsWithin(done, time.Second))
}

func TestSessSemaphoreMinimumLimit(t *testing.T) {
	ss := NewSessSemaphore(0)
	assert.True(t, returnsWithin(acquireAsync(ss), time.Second), "a zero limit must still allow one holder")

	ss.SetLimit(-5)
	ss.Release()
	assert.True(t, returnsWithin(acquireAsync(ss), time.Second), "a negative limit must still allow one holder")
}
