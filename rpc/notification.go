// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package rpc

import (
	"errors"
	"sync"

	"golang.org/x/net/context"
)

var (
	// ErrNotificationsUnsupported is returned when the connection doesn't support notifications
	ErrNotificationsUnsupported = errors.New("subscription notifications not supported by the current transport")
	// ErrNotificationNotFound is returned when the notification for the given id is not found
	ErrNotificationNotFound = errors.New("notification not found")
	// errNotifierStopped is returned when the notifier is stopped (e.g. codec is closed)
	errNotifierStopped = errors.New("unable to send notification")
	// errNotificationQueueFull is returns when there are too many notifications in the queue
	errNotificationQueueFull = errors.New("too many pending notifications")
)

// UnsubscribeCallback defines a callback that is called when a subcription ends.
// It receives the subscription id as argument.
type UnsubscribeCallback func(SubscriptionID)

type notifierKey struct{}

// NotifierFromContext returns the Notifier value stored in ctx, if any.
func NotifierFromContext(ctx context.Context) (*Notifier, bool) {
	n, ok := ctx.Value(notifierKey{}).(*Notifier)
	return n, ok
}

// SubscriptionID defines the subscript ID type
type SubscriptionID string

// bufferedNotifier is a notifier that queues notifications in an internal queue and
// send them as fast as possible to the client from this queue. It will stop if the
// queue grows past a given size.
type Notifier struct {
	codec      ServerCodec  // underlying connection
	activateMu sync.RWMutex // guards the inactive and active maps
	inactive   map[SubscriptionID]UnsubscribeCallback
	active     map[SubscriptionID]UnsubscribeCallback
}

// newBufferedNotifier returns a notifier that queues notifications in an internal queue
// from which notifications are send as fast as possible to the client. If the queue size
// limit is reached (client is unable to keep up) it will stop and closes the codec.
func newNotifier(codec ServerCodec, size int) *Notifier {
	return &Notifier{
		codec:    codec,
		inactive: make(map[SubscriptionID]UnsubscribeCallback),
		active:   make(map[SubscriptionID]UnsubscribeCallback),
	}
}

// NewSubscription creates a new subscription. The subscription is enabled by the RPC
// server after the subscription ID was send to the client. All notifications before the
// subscription was activated are dropped. This prevents that the client receives
// notifications for a subscription before the subscription ID was received.
func (n *Notifier) NewSubscription(cb UnsubscribeCallback) (SubscriptionID, error) {
	subid, err := newSubscriptionID()
	if err == nil {
		// subscriptions are inactive until the ID has been send to the
		// client. The RPC server will call the Activate function to mark
		// a subscription as active.
		n.activateMu.Lock()
		n.inactive[subid] = cb
		n.activateMu.Unlock()
	}
	return subid, err
}

// enable marks an inactive subscription as active. Notifications are only send
// for active subscriptions. Notifications for inactive subscriptions are dropped.
func (n *Notifier) enable(id SubscriptionID) {
	n.activateMu.Lock()
	if cb, found := n.inactive[id]; found {
		n.active[id] = cb
		delete(n.inactive, id)
	}
	n.activateMu.Unlock()
}

// Closed returns a channel that is closed when the connection is closed.
func (n *Notifier) Closed() <-chan interface{} {
	return n.codec.Closed()
}

// Unsubscribe a subscription. If the given subscription could not be found
// ErrNotificationNotFound is returned.
func (n *Notifier) Unsubscribe(id SubscriptionID) error {
	n.activateMu.Lock()
	defer n.activateMu.Unlock()
	if _, found := n.active[id]; !found {
		return ErrNotificationNotFound
	}
	delete(n.active, id)
	return nil
}

// Send a notification to the client. If the subscription was not active
// the notification is dropped without returning an error.
func (n *Notifier) Send(id SubscriptionID, event interface{}) error {
	n.activateMu.RLock()
	_, active := n.active[id]
	n.activateMu.RUnlock()

	// subscription id has not been send to the client yet, drop event
	if active {
		msg := n.codec.CreateNotification(string(id), event)
		if err := n.codec.Write(msg); err != nil {
			n.codec.Close()
			return err
		}
	}
	return nil
}
