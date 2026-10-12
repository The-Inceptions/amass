// Copyright © by Jeff Foley 2017-2026. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.
// SPDX-License-Identifier: Apache-2.0

package pubsub

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func receive(t *testing.T, ch <-chan *string) (string, bool) {
	t.Helper()

	select {
	case msg, ok := <-ch:
		if !ok {
			return "", false
		}
		return *msg, true
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for a log message")
	}
	return "", false
}

func TestLoggerDeliversToSubscribers(t *testing.T) {
	l := NewLogger()
	defer l.Close()

	a := l.Subscribe()
	b := l.Subscribe()
	l.Publish("hello")

	for _, ch := range []<-chan *string{a, b} {
		msg, ok := receive(t, ch)
		require.True(t, ok)
		assert.Equal(t, "hello", msg)
	}
}

func TestLoggerConcurrentPublishAndSubscribe(t *testing.T) {
	l := NewLogger()
	defer l.Close()

	var wg sync.WaitGroup
	for i := range 8 {
		wg.Add(2)
		go func() {
			defer wg.Done()
			for j := range 100 {
				l.Publish(fmt.Sprintf("%d-%d", i, j))
			}
		}()
		go func() {
			defer wg.Done()
			l.Unsubscribe(l.Subscribe())
		}()
	}
	wg.Wait()
}

func TestLoggerSlowSubscriberDoesNotLeakOrPanic(t *testing.T) {
	l := NewLogger()
	_ = l.Subscribe() // never read from

	before := runtime.NumGoroutine()
	for i := range 5 * subscriberBuffer {
		l.Publish(fmt.Sprintf("message %d", i))
	}
	// give the broadcast goroutine time to work through the queue
	time.Sleep(200 * time.Millisecond)
	assert.LessOrEqual(t, runtime.NumGoroutine(), before+1, "a full subscriber must not leave goroutines behind")

	assert.NotPanics(t, l.Close)
}

func TestLoggerUnsubscribeStopsDelivery(t *testing.T) {
	l := NewLogger()
	defer l.Close()

	keep := l.Subscribe()
	gone := l.Subscribe()
	l.Unsubscribe(gone)

	_, ok := <-gone
	assert.False(t, ok, "Unsubscribe must close the channel")

	l.Publish("after")
	msg, ok := receive(t, keep)
	require.True(t, ok)
	assert.Equal(t, "after", msg)

	assert.NotPanics(t, func() { l.Unsubscribe(gone) }, "unsubscribing twice must be safe")
}

func TestLoggerCloseIsIdempotent(t *testing.T) {
	l := NewLogger()
	sub := l.Subscribe()

	l.Close()
	assert.NotPanics(t, l.Close)

	_, ok := <-sub
	assert.False(t, ok, "Close must close subscribed channels")

	late := l.Subscribe()
	_, ok = <-late
	assert.False(t, ok, "a subscription made after Close must come back closed")

	assert.NotPanics(t, func() {
		l.Publish("ignored")
		l.Unsubscribe(sub)
		l.Unsubscribe(late)
	})
}
