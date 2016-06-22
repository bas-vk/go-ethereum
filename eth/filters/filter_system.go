// Copyright 2015 The go-ethereum Authors
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

// Package filters implements an ethereum filtering system for block,
// transactions and log events.
package filters

import (
	"bufio"
	crand "crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

const (
	// UnknownSubscription indicates an unkown subscription type
	UnknownSubscription Type = iota
	// ChainEventSubscription queries for new blocks
	ChainEventSubscription
	// PendingTxSubscription queries pending transactions
	PendingTxSubscription
	// LogSubscription queries for new or removed (chain reorg) logs
	LogSubscription
	// PendingLogSubscription queries for logs for the pending block
	PendingLogSubscription
)

var (
	filterIDGenMu sync.Mutex
	filterIDGen   = suscriptionIDGenerator()

	ErrInvalidSubscriptionID = errors.New("unable to parse id")
)

// Type determines the kind of filter and is used to put the filter in to
// the correct bucket when added.
type Type byte

// SubscriptionID determines the type for a subscription identifier.
type SubscriptionID [16]byte

// MarshalJSON serializes a SubscriptionID into its JSON representation.
func (f SubscriptionID) MarshalJSON() ([]byte, error) {
	return json.Marshal(fmt.Sprintf("0x%x", f))
}

// UnmarshalJSON parses a SubscriptionID from its JSON representation.
func (f *SubscriptionID) UnmarshalJSON(data []byte) error {
	// `"0x...."`
	if len(data) != 36 || data[0] != '"' || data[35] != '"' || data[1] != '0' || (data[2] != 'x' && data[2] != 'X') {
		return ErrInvalidSubscriptionID
	}

	_, err := hex.Decode(f[:], data[3:35])
	return err
}

// Log is a helper that can hold additional information about vm.Log
// necessary for the RPC interface.
type Log struct {
	*vm.Log
	Removed bool `json:"removed"`
}

// suscriptionIDGenerator helper utility that generates a (pseudo) random sequence of bytes
// that are used to generate filter identifiers.
func suscriptionIDGenerator() *rand.Rand {
	if seed, err := binary.ReadVarint(bufio.NewReader(crand.Reader)); err == nil {
		return rand.New(rand.NewSource(seed))
	}
	return rand.New(rand.NewSource(int64(time.Now().Nanosecond())))
}

// newID generates a subscription identifier.
func newID() SubscriptionID {
	filterIDGenMu.Lock()
	defer filterIDGenMu.Unlock()

	var id SubscriptionID
	for i := 0; i < len(id); i += 7 {
		val := filterIDGen.Int63()
		for j := 0; i+j < len(id) && j < 7; j++ {
			id[i+j] = byte(val)
			val >>= 8
		}
	}

	return id
}

type filter struct {
	id          SubscriptionID
	typ         Type
	created     time.Time
	uninstalled chan struct{}
	hashes      chan<- common.Hash // results in case filter type returns hashes
	blocks      chan<- *types.Block
	logsCrit    FilterCriteria
	logs        chan<- []Log // results in case filter type returns logs
}

// Manager creates subscriptions, processes events and broadcasts them to the
// subscription which match the subscription criteria.
type Manager struct {
	sub event.Subscription // event mux subscription, closed when the manager is issued to stop

	install   chan *filter // install filter for event notification
	uninstall chan *filter // remove filter for event notification

	subscriptionMu sync.RWMutex                     // guard subscription
	subscriptions  map[SubscriptionID]*Subscription // all installed filters
}

// Subscription is created when the client registers itself for a particular event.
type Subscription struct {
	ID  SubscriptionID
	f   *filter
	m   *Manager
	err chan error
}

// Err the last error that occurred. When the subscription is cancelled the
// returned channel is closed.
func (s *Subscription) Err() <-chan error {
	return s.err
}

// Cancel stops and deletes the subscription.
// It will cause the Err() chan to return nil once.
func (s *Subscription) Cancel() {
	s.m.uninstall <- s.f // send uninstall request to manager work loop

	s.m.subscriptionMu.Lock()
	delete(s.m.subscriptions, s.ID)
	s.m.subscriptionMu.Unlock()

	// wait for filter to be uninstalled in work loop before returning
	// this ensures that the manager won't use the event channel which
	// will probably be closed by the client asap after this method returns.
	<-s.f.uninstalled

	close(s.err)
}

// NewManager creates a new manager that listens for event on the given mux,
// parses and filters them. It uses the all map to retrieve filter changes. The
// work loop holds its own index that is used to forward events to filters.
//
// The returned manager has a loop that needs to be stopped with the Stop function
// or by stopping the given mux.
func NewManager(mux *event.TypeMux) *Manager {
	sub := mux.Subscribe(
		core.PendingLogsEvent{},
		core.RemovedLogsEvent{},
		core.ChainEvent{},
		core.TxPreEvent{},
		vm.Logs(nil),
	)

	m := &Manager{
		sub:           sub,
		install:       make(chan *filter),
		uninstall:     make(chan *filter),
		subscriptions: make(map[SubscriptionID]*Subscription),
	}

	go m.run()

	return m
}

func (m *Manager) subscribe(f *filter) (*Subscription, error) {
	m.install <- f

	s := &Subscription{f.id, f, m, make(chan error)}

	m.subscriptionMu.Lock()
	m.subscriptions[f.id] = s
	m.subscriptionMu.Unlock()

	return s, nil
}

// SubscribeChainEvent will send block hashes for block that are added onto,
// or replacing the current chain head to the blockHashes channel until the
// subscription is closed, or an error occures.
func (m *Manager) SubscribeChainEvent(blocks chan<- *types.Block) (*Subscription, error) {
	f := &filter{
		id:          newID(),
		typ:         ChainEventSubscription,
		created:     time.Now(),
		blocks:      blocks,
		uninstalled: make(chan struct{}),
	}

	return m.subscribe(f)
}

func (m *Manager) SubscribeLogs(crit FilterCriteria, logs chan<- []Log) (*Subscription, error) {
	f := &filter{
		id:          newID(),
		typ:         LogSubscription,
		logsCrit:    crit,
		created:     time.Now(),
		logs:        logs,
		uninstalled: make(chan struct{}),
	}

	return m.subscribe(f)
}

// SubscriptionType returns the filter type for the given id.
// If the subscription could not be found UnknownSubscription is returned.
func (m *Manager) SubscriptionType(id SubscriptionID) Type {
	m.subscriptionMu.RLock()
	defer m.subscriptionMu.RUnlock()

	if sub, found := m.subscriptions[id]; found {
		return sub.f.typ
	}
	return UnknownSubscription
}

type filterIndex map[Type]map[SubscriptionID]*filter

// broadcast event to filters that match criteria.
func broadcast(filters filterIndex, ev *event.Event) {
	switch e := ev.Data.(type) {
	case core.ChainEvent:
		for _, f := range filters[ChainEventSubscription] {
			if ev.Time.After(f.created) {
				f.blocks <- e.Block
			}
		}
	case core.TxPreEvent:
		for _, f := range filters[PendingTxSubscription] {
			if ev.Time.After(f.created) {
				f.hashes <- e.Tx.Hash()
			}
		}
	case vm.Logs:
		if len(e) > 0 {
			for _, f := range filters[LogSubscription] {
				if ev.Time.After(f.created) {
					if matchedLogs := filterLogs(convertLogs(e, false), f.logsCrit.Addresses, f.logsCrit.Topics); len(matchedLogs) > 0 {
						f.logs <- matchedLogs
					}
				}
			}
		}
	case core.RemovedLogsEvent:
		for _, f := range filters[LogSubscription] {
			if ev.Time.After(f.created) {
				if matchedLogs := filterLogs(convertLogs(e.Logs, true), f.logsCrit.Addresses, f.logsCrit.Topics); len(matchedLogs) > 0 {
					f.logs <- matchedLogs
				}
			}
		}
	case core.PendingLogsEvent:
		for _, f := range filters[PendingLogSubscription] {
			if ev.Time.After(f.created) {
				if matchedLogs := filterLogs(convertLogs(e.Logs, false), f.logsCrit.Addresses, f.logsCrit.Topics); len(matchedLogs) > 0 {
					f.logs <- matchedLogs
				}
			}
		}
	default:
		glog.V(logger.Debug).Infof("unsupported event: %T", e)
	}
}

func (m *Manager) run() {
	index := make(filterIndex)

	for {
		select {
		case ev, active := <-m.sub.Chan():
			if !active { // stop issued
				glog.V(logger.Debug).Infoln("filter manager stopped")
				return
			}
			broadcast(index, ev)
		case f := <-m.install:
			if _, found := index[f.typ]; !found {
				index[f.typ] = make(map[SubscriptionID]*filter)
			}
			index[f.typ][f.id] = f
		case f := <-m.uninstall:
			delete(index[f.typ], f.id)
			close(f.uninstalled)
		}
	}
}

/*

// NewLogFilter returns a filter identifier that can be used to fetch logs matching
// the given criteria.
func (m *Manager) NewLogFilter(crit FilterCriteria, cb logsCallback) (FilterID, error) {
	id := newID()

	f := &filter{
		ID:         id,
		created:    time.Now(),
		lastUsed:   time.Now(),
		canTimeout: true,
		typ:        LogFilter,
		hashes:     make(chan common.Hash, maxPendingHashes),
		logs:       make(chan []Log, maxPendingLogs),
		lc:         cb,
		logsCrit:   crit,
	}

	m.allMu.Lock()
	m.all[id] = f
	m.allMu.Unlock()

	m.install <- f

	return id, nil
}

// GetLogFilterCriteria returns the filtering criteria for a filter, or an error in
// case the filter could not be found.
func (m *Manager) GetLogFilterCriteria(id FilterID) (FilterCriteria, error) {
	m.allMu.RLock()
	defer m.allMu.RUnlock()

	if f, found := m.all[id]; found {
		return f.logsCrit, nil
	}

	return FilterCriteria{}, errFilterNotFound
}

// NewPendingLogFilter creates a filter that returns new pending logs that match the given criteria.
func (m *Manager) NewPendingLogFilter(crit FilterCriteria, cb logsCallback) (FilterID, error) {
	id := newID()

	f := &filter{
		ID:         id,
		created:    time.Now(),
		lastUsed:   time.Now(),
		canTimeout: true,
		typ:        PendingLogFilter,
		hashes:     make(chan common.Hash, maxPendingHashes),
		logs:       make(chan []Log, maxPendingLogs),
		lc:         cb,
		logsCrit:   crit,
	}

	m.allMu.Lock()
	m.all[id] = f
	m.allMu.Unlock()

	m.install <- f

	return id, nil
}

// GetPendingLogFilterChanges returns logs for the pending block.
func (m *Manager) GetPendingLogFilterChanges(id FilterID) ([]Log, error) {
	m.allMu.RLock()
	defer m.allMu.RUnlock()

	if f, found := m.all[id]; found && f.typ == PendingLogFilter {
		f.lastUsed = time.Now()
		allLogs := make([]Log, 0, len(f.logs)) // prevent (most) allocs
		for {
			select {
			case logs := <-f.logs:
				allLogs = append(allLogs, logs...)
			default: // available logs read
				return allLogs, nil
			}
		}
	}

	return nil, errFilterNotFound
}

// GetLogFilterChanges returns all logs matching the criteria for the filter with the given filter id.
func (m *Manager) GetLogFilterChanges(id FilterID) ([]Log, error) {
	m.allMu.RLock()
	defer m.allMu.RUnlock()

	if f, found := m.all[id]; found && f.typ == LogFilter {
		f.lastUsed = time.Now()
		allLogs := make([]Log, 0, len(f.logs)) // prevent (most) allocs for the append
		for {
			select {
			case logs := <-f.logs:
				allLogs = append(allLogs, logs...)
			default: // available logs read
				return allLogs, nil
			}
		}
	}

	return nil, errFilterNotFound
}

// NewPendingTransactionFilter creates a filter that retrieves pending transactions.
func (m *Manager) NewPendingTransactionFilter() (FilterID, error) {
	id := newID()

	f := &filter{
		ID:         id,
		created:    time.Now(),
		lastUsed:   time.Now(),
		canTimeout: true,
		typ:        PendingTxFilter,
		hashes:     make(chan common.Hash, maxPendingHashes),
		logs:       make(chan []Log, maxPendingLogs),
	}

	m.allMu.Lock()
	m.all[id] = f
	m.allMu.Unlock()

	m.install <- f

	return id, nil
}

// GetPendingTxFilterChanges returns hashes for pending transactions which are added since the last poll.
func (m *Manager) GetPendingTxFilterChanges(id FilterID) ([]common.Hash, error) {
	m.allMu.RLock()
	defer m.allMu.RUnlock()

	if f, found := m.all[id]; found && f.typ == PendingTxFilter {
		f.lastUsed = time.Now()
		hashes := make([]common.Hash, 0, len(f.hashes)) // prevent (most) allocs
		for {
			select {
			case hash := <-f.hashes:
				hashes = append(hashes, hash)
			default: // read available tx hashes
				return hashes, nil
			}
		}
	}

	return nil, errFilterNotFound
}

type filterIndex map[Type]map[FilterID]*filter

// process an event and forward to filters that match the criteria.
func (m *Manager) process(filters filterIndex, ev *event.Event) {
	var inactive []*filter

	logHandler := func(f *filter, logs []Log) {
		if f.lc != nil && len(logs) > 0 {
			f.lc(f.ID, logs)
		} else if len(logs) > 0 {
			select {
			case f.logs <- logs:
				return
			default: // data queue full, disable filter
				inactive = append(inactive, f)
			}
		}
	}

	switch e := ev.Data.(type) {
	case core.ChainEvent:
		for _, f := range filters[BlockFilter] {
			if ev.Time.After(f.created) {
				select {
				case f.hashes <- e.Hash:
					continue
				default:
					// data queue full, disable filter
					inactive = append(inactive, f)
				}
			}
		}
	case core.TxPreEvent:
		for _, f := range filters[PendingTxFilter] {
			if ev.Time.After(f.created) {
				select {
				case f.hashes <- e.Tx.Hash():
					continue
				default:
					// data queue full, disable filter
					inactive = append(inactive, f)
				}
			}
		}
	case vm.Logs:
		for _, f := range filters[LogFilter] {
			if ev.Time.After(f.created) {
				matchedLogs := filterLogs(convertLogs(e, false), f.logsCrit.Addresses, f.logsCrit.Topics)
				logHandler(f, matchedLogs)
			}
		}
	case core.RemovedLogsEvent:
		for _, f := range filters[LogFilter] {
			if ev.Time.After(f.created) {
				matchedLogs := filterLogs(convertLogs(e.Logs, true), f.logsCrit.Addresses, f.logsCrit.Topics)
				logHandler(f, matchedLogs)
			}
		}
	case core.PendingLogsEvent:
		for _, f := range filters[PendingLogFilter] {
			if ev.Time.After(f.created) {
				matchedLogs := filterLogs(convertLogs(e.Logs, false), f.logsCrit.Addresses, f.logsCrit.Topics)
				logHandler(f, matchedLogs)
			}
		}
	}

	m.allMu.Lock()
	for _, f := range inactive {
		delete(m.all, f.ID)
		glog.Warningf("filter 0x%x uninstalled, queue full\n", f.ID)
	}
	m.allMu.Unlock()

	// remove filter for event listening, this must be run in a seperate go routine
	// since timeout is called from the work loop and sending uninstall requests to the
	// work loop from "itself" may deadlock when the uninstall channel is full.
	go func() {
		for _, f := range inactive {
			m.uninstall <- f
		}
	}()
}

// timeout uninstalls all filters that have not been used in the last 5 minutes.
func (m *Manager) timeout() {
	deadline := time.Now().Add(-5 * time.Minute)
	var inactive []*filter
	m.allMu.Lock()
	for _, f := range m.all {
		if f.lastUsed.Before(deadline) && f.canTimeout {
			delete(m.all, f.ID) // filter cannot be used from the external
			inactive = append(inactive, f)
		}
	}
	m.allMu.Unlock()

	// remove filter for event listening, this must be run in a seperate go routine
	// since timeout is called from the work loop and sending uninstall requests to the
	// work loop from "itself" may deadlock when the uninstall channel is full.
	go func() {
		for _, f := range inactive {
			m.uninstall <- f // delete for the internal work loop
		}
	}()
}

// run is the manager work loop.
// It will receive events and forwards them to installed filters.
// Inactive filters will be uninstalled.
func (m *Manager) run() {
	index := make(filterIndex)
	timeout := time.NewTicker(30 * time.Second)

	for {
		select {
		case f := <-m.install:
			// lazy load
			if _, found := index[f.typ]; !found {
				index[f.typ] = make(map[FilterID]*filter)
			}
			index[f.typ][f.ID] = f
		case f := <-m.uninstall:
			close(f.hashes)
			close(f.logs)
			delete(index[f.typ], f.ID)
		case ev, ok := <-m.sub.Chan():
			if !ok {
				glog.V(logger.Debug).Infoln("filter manager stopped")
				return
			}
			m.process(index, ev)
		case <-timeout.C:
			m.timeout()
		}
	}
}

*/

// Stop the filter system.
func (m *Manager) Stop() {
	m.sub.Unsubscribe() // end worker loop
}

// convertLogs is a helper utility that converts vm.Logs to []filter.Log.
func convertLogs(in vm.Logs, removed bool) []Log {
	logs := make([]Log, len(in))
	for i, l := range in {
		logs[i] = Log{l, false}
	}
	return logs
}
