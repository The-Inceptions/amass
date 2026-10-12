// Copyright © by Jeff Foley 2017-2026. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.
// SPDX-License-Identifier: Apache-2.0

package pubsub

import (
	"sync"

	"github.com/caffix/queue"
)

// subscriberBuffer is how far a subscriber can fall behind before messages are dropped for it.
const subscriberBuffer = 1000

// Logger provides functionality for logging and subscribing to logs.
type Logger struct {
	sync.Mutex
	done   chan struct{}                   // Channel to signal closure of the logger.
	q      queue.Queue                     // Embeds a queue for managing log messages.
	closed bool                            // Set once the logger has been closed.
	subs   map[<-chan *string]chan *string // Channels used to communicate log messages.
}

// NewLogger initializes and returns a new instance of Logger.
func NewLogger() *Logger {
	l := &Logger{
		done: make(chan struct{}),
		q:    queue.NewQueue(),
		subs: make(map[<-chan *string]chan *string),
	}
	go l.broadcastMessages()
	return l
}

// Publish sends a log message to the logger.
func (l *Logger) Publish(msg string) {
	l.Lock()
	defer l.Unlock()

	if !l.closed && len(l.subs) > 0 {
		l.q.Append(msg)
	}
}

// Write allows the Logger to be used as a Writer and in structured logging.
func (l *Logger) Write(p []byte) (n int, err error) {
	l.Publish(string(p))
	return len(p), nil
}

// Subscribe provides a read-only channel to receive log messages.
// This allows external components to "listen" for new logs.
// The channel is closed by Unsubscribe or when the logger is closed.
func (l *Logger) Subscribe() <-chan *string {
	l.Lock()
	defer l.Unlock()

	ch := make(chan *string, subscriberBuffer)
	if l.closed {
		close(ch)
		return ch
	}

	l.subs[ch] = ch
	return ch
}

// Unsubscribe stops delivery to a channel returned by Subscribe and closes it.
func (l *Logger) Unsubscribe(sub <-chan *string) {
	l.Lock()
	defer l.Unlock()

	if ch, found := l.subs[sub]; found {
		delete(l.subs, sub)
		close(ch)
	}
}

// Close shuts down the logger and closes all subscribed channels. It is safe to call more than once.
func (l *Logger) Close() {
	l.Lock()
	if l.closed {
		l.Unlock()
		return
	}
	l.closed = true
	close(l.done)

	// Close all subscribed channels to signal that no more messages will be sent.
	for sub, ch := range l.subs {
		delete(l.subs, sub)
		close(ch)
	}
	l.Unlock()

	// drain the queue
	l.q.Process(func(any) {})
}

func (l *Logger) broadcastMessages() {
	for {
		select {
		case <-l.done:
			return
		case <-l.q.Signal():
		}

		l.q.Process(l.msgToSubscribers)
	}
}

func (l *Logger) msgToSubscribers(e any) {
	msg, ok := e.(string)
	if !ok {
		return
	}

	l.Lock()
	defer l.Unlock()

	if l.closed {
		return
	}

	for _, ch := range l.subs {
		select {
		case ch <- &msg:
		default:
			// the subscriber has fallen too far behind, so it misses this message
		}
	}
}
