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

package filters

import (
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/event"
)

var (
	mux           = new(event.TypeMux)
	filterManager = NewManager(mux)
)

// TestFilterIdSerialization tests if FilterIs is correct serialized and parsed.
func TestFilterIdSerialization(t *testing.T) {
	t.Parallel()

	id := newID()

	serialized, err := json.Marshal(id)
	if err != nil {
		t.Fatal(err)
	}

	if len(serialized) != 36 {
		t.Fatalf("invalid filter id length (%s), want %d, got %d", serialized, 36, len(serialized))
	}

	var filterID SubscriptionID
	if err := json.Unmarshal(serialized, &filterID); err != nil {
		t.Fatal(err)
	}

	if filterID != id {
		t.Errorf("invalid filter id, want %x, got %x", id, filterID)
	}
}

// TestBlockFilter tests if a block filter callback is called when core.ChainEvent are posted.
// It creates multiple filters:
// - one at the start and should receive all posted chain events and a second (blockHashes)
// - one that is created after a cutoff moment and uninstalled after a second cutoff moment (blockHashes[cutoff1:cutoff2])
// - one that is created after the second cutoff moment (blockHashes[cutoff2:])
func TestBlockFilter(t *testing.T) {
	/*
		t.Parallel()

		var (
			chainHeadEvents = []core.ChainHeadEvent{
				{Hash: common.HexToHash("0xd4e56740f876aef8c010b86a40d5f56745a118d0906a34e69aec8c0db1cb8fa3")},
				{Hash: common.HexToHash("0x88e96d4537bea4d9c05d12549907b32561d3bf31f45aae734cdc119f13406cb6")},
				{Hash: common.HexToHash("0xb495a1d7e6663152ae92708da4843337b958146015a2802f4193a410044698c9")},
				{Hash: common.HexToHash("0x3d6122660cc824376f11ee842f83addc3525e2dd6756b9bcf0affa6aa88cf741")},
				{Hash: common.HexToHash("0x23adf5a3be0f5235b36941bcb29b62504278ec5b9cdfa277b992ba4a7a3cd3a2")},
				{Hash: common.HexToHash("0xf37c632d361e0a93f08ba29b1a2c708d9caa3ee19d1ee8d2a02612bffe49f0a9")},
				{Hash: common.HexToHash("0x1f1aed8e3694a067496c248e61879cda99b0709a1dfbacd0b693750df06b326e")},
				{Hash: common.HexToHash("0xe0c7c0b46e116b874354dce6f64b8581bd239186b03f30a978e3dc38656f723a")},
				{Hash: common.HexToHash("0x2ce94342df186bab4165c268c43ab982d360c9474f429fec5565adfc5d1f258b")},
				{Hash: common.HexToHash("0x997e47bf4cac509c627753c06385ac866641ec6f883734ff7944411000dc576e")},
			}
		)

		chan0 := make(chan *types.Block)
		sub0, err := filterManager.SubscribeChainHead(chan0)
		if err != nil {
			t.Fatal(err)
		}

		chan1 := make(chan *types.Block)
		sub1, err := filterManager.SubscribeChainHead(chan1)
		if err != nil {
			t.Fatal(err)
		}

		go func() { // simulate client
			time.Sleep(2 * time.Second)

			i1, i2 := 0, 0
			for i1 != len(chainHeadEvents) || i2 != len(chainHeadEvents) {
				select {
				case block := <-chan0:
					if chainHeadEvents[i1].Hash != block.Hash() {
						t.Errorf("sub0 received invalid hash on index %d, want %x, got %x", i1, chainHeadEvents[i1].Hash, block)
					}
					i1++
				case block := <-chan1:
					if chainHeadEvents[i2].Hash != block.Hash() {
						t.Errorf("sub0 received invalid hash on index %d, want %x, got %x", i2, chainHeadEvents[i2].Hash, block)
					}
					i2++
				}
			}

			sub0.Cancel()
			sub1.Cancel()
		}()

		for _, e := range chainHeadEvents {
			mux.Post(e)
		}

		if err := <-sub0.Err(); err != nil {
			t.Errorf("sub0 expected nil, got %v", err)
		}

		if err := <-sub1.Err(); err != nil {
			t.Errorf("sub1 expected nil, got %v", err)
		}
	*/
}

/*
// TestPendingTxFilter tests whether pending tx filters retrieve all pending transactions that are posted to the event mux.
func TestPendingTxFilter(t *testing.T) {
	t.Parallel()

	var (
		transactions = []*types.Transaction{
			types.NewTransaction(0, common.HexToAddress("0xb794f5ea0ba39494ce83a213fffba74279579268"), new(big.Int), new(big.Int), new(big.Int), nil),
			types.NewTransaction(1, common.HexToAddress("0xb794f5ea0ba39494ce83a213fffba74279579268"), new(big.Int), new(big.Int), new(big.Int), nil),
			types.NewTransaction(2, common.HexToAddress("0xb794f5ea0ba39494ce83a213fffba74279579268"), new(big.Int), new(big.Int), new(big.Int), nil),
			types.NewTransaction(3, common.HexToAddress("0xb794f5ea0ba39494ce83a213fffba74279579268"), new(big.Int), new(big.Int), new(big.Int), nil),
			types.NewTransaction(4, common.HexToAddress("0xb794f5ea0ba39494ce83a213fffba74279579268"), new(big.Int), new(big.Int), new(big.Int), nil),
		}

		hashes []common.Hash
	)

	fid1, err := filterManager.NewPendingTransactionFilter()
	if err != nil {
		t.Fatal(err)
	}

	for _, tx := range transactions {
		ev := core.TxPreEvent{Tx: tx}
		mux.Post(ev)
	}

	for {
		h, err := filterManager.GetPendingTxFilterChanges(fid1)
		if err != nil {
			t.Fatalf("unable to fetch pending transactions: %v", err)
		}
		hashes = append(hashes, h...)

		if len(hashes) >= len(transactions) {
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	for i := range hashes {
		if hashes[i] != transactions[i].Hash() {
			t.Errorf("hashes[%d] invalid, want %x, got %x", i, transactions[i].Hash(), hashes[i])
		}
	}
}

// TestLogFilter tests whether log filters match the correct logs that are posted to the event mux.
func TestLogFilter(t *testing.T) {
	t.Parallel()

	var (
		err error

		firstAddr      = common.HexToAddress("0x1111111111111111111111111111111111111111")
		secondAddr     = common.HexToAddress("0x2222222222222222222222222222222222222222")
		thirdAddress   = common.HexToAddress("0x3333333333333333333333333333333333333333")
		notUsedAddress = common.HexToAddress("0x9999999999999999999999999999999999999999")
		firstTopic     = common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
		secondTopic    = common.HexToHash("0x2222222222222222222222222222222222222222222222222222222222222222")
		notUsedTopic   = common.HexToHash("0x9999999999999999999999999999999999999999999999999999999999999999")

		allLogs = vm.Logs{
			// Note, these are used for comparison of the test cases.
			0: vm.NewLog(firstAddr, []common.Hash{}, []byte(""), 0),
			1: vm.NewLog(firstAddr, []common.Hash{firstTopic}, []byte(""), 1),
			2: vm.NewLog(secondAddr, []common.Hash{firstTopic}, []byte(""), 1),
			3: vm.NewLog(thirdAddress, []common.Hash{secondTopic}, []byte(""), 2),
			4: vm.NewLog(thirdAddress, []common.Hash{secondTopic}, []byte(""), 3),
		}

		testCases = []struct {
			crit     FilterCriteria
			expected vm.Logs
			id       FilterID
		}{
			// match all
			0: {FilterCriteria{}, allLogs, FilterID{}},
			// match none due to no matching addresses
			1: {FilterCriteria{Addresses: []common.Address{common.Address{}, notUsedAddress}, Topics: [][]common.Hash{allLogs[0].Topics}}, vm.Logs{}, FilterID{}},
			// match logs based on addresses, ignore topics
			2: {FilterCriteria{Addresses: []common.Address{firstAddr}}, allLogs[:2], FilterID{}},
			// match none due to no matching topics (match with address)
			3: {FilterCriteria{Addresses: []common.Address{secondAddr}, Topics: [][]common.Hash{[]common.Hash{notUsedTopic}}}, vm.Logs{}, FilterID{}},
			// match logs based on addresses and topics
			4: {FilterCriteria{Addresses: []common.Address{thirdAddress}, Topics: [][]common.Hash{[]common.Hash{firstTopic, secondTopic}}}, allLogs[3:5], FilterID{}},
			// match logs based on multiple addresses and "or" topics
			5: {FilterCriteria{Addresses: []common.Address{secondAddr, thirdAddress}, Topics: [][]common.Hash{[]common.Hash{firstTopic, secondTopic}}}, allLogs[2:5], FilterID{}},
			// block numbers are ignored for filters created with New***Filter, these return all logs that match the given criterias when the state changes
			6: {FilterCriteria{Addresses: []common.Address{firstAddr}, FromBlock: rpc.BlockNumber(1), ToBlock: rpc.BlockNumber(2)}, allLogs[:2], FilterID{}},
		}
	)

	// create all filters
	for i := range testCases {
		testCases[i].id, err = filterManager.NewLogFilter(testCases[i].crit, nil)
		if err != nil {
			t.Fatal(err)
		}
	}

	// raise events
	if err = mux.Post(allLogs); err != nil {
		t.Fatal(err)
	}

	for i, tt := range testCases {
		var fetched []Log
		for { // fetch all expected logs
			logs, err := filterManager.GetLogFilterChanges(tt.id)
			if err != nil {
				t.Errorf("unable to retrieve logs for case %d, %s", i, err)
				break
			}

			fetched = append(fetched, logs...)
			if len(fetched) >= len(tt.expected) {
				break
			}

			time.Sleep(100 * time.Millisecond)
		}

		if len(fetched) != len(tt.expected) {
			t.Errorf("invalid number of logs for case %d, want %d log(s), got %d", i, len(tt.expected), len(fetched))
			return
		}

		for l := range fetched {
			if fetched[l].Removed {
				t.Errorf("expected log not to be removed for log %d in case %d", l, i)
			}
			if !reflect.DeepEqual(fetched[l].Log, tt.expected[l]) {
				t.Errorf("invalid log on index %d for case %d", l, i)
			}

		}
	}
}

func TestPendingLogFilter(t *testing.T) {
	t.Parallel()

	var (
		firstAddr      = common.HexToAddress("0x1111111111111111111111111111111111111111")
		secondAddr     = common.HexToAddress("0x2222222222222222222222222222222222222222")
		thirdAddress   = common.HexToAddress("0x3333333333333333333333333333333333333333")
		notUsedAddress = common.HexToAddress("0x9999999999999999999999999999999999999999")
		firstTopic     = common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
		secondTopic    = common.HexToHash("0x2222222222222222222222222222222222222222222222222222222222222222")
		thirdTopic     = common.HexToHash("0x3333333333333333333333333333333333333333333333333333333333333333")
		forthTopic     = common.HexToHash("0x4444444444444444444444444444444444444444444444444444444444444444")
		notUsedTopic   = common.HexToHash("0x9999999999999999999999999999999999999999999999999999999999999999")

		allLogs = []core.PendingLogsEvent{
			0: core.PendingLogsEvent{Logs: vm.Logs{vm.NewLog(firstAddr, []common.Hash{}, []byte(""), 0)}},
			1: core.PendingLogsEvent{Logs: vm.Logs{vm.NewLog(firstAddr, []common.Hash{firstTopic}, []byte(""), 1)}},
			2: core.PendingLogsEvent{Logs: vm.Logs{vm.NewLog(secondAddr, []common.Hash{firstTopic}, []byte(""), 2)}},
			3: core.PendingLogsEvent{Logs: vm.Logs{vm.NewLog(thirdAddress, []common.Hash{secondTopic}, []byte(""), 3)}},
			4: core.PendingLogsEvent{Logs: vm.Logs{vm.NewLog(thirdAddress, []common.Hash{secondTopic}, []byte(""), 4)}},
			5: core.PendingLogsEvent{Logs: vm.Logs{
				vm.NewLog(thirdAddress, []common.Hash{firstTopic}, []byte(""), 5),
				vm.NewLog(thirdAddress, []common.Hash{thirdTopic}, []byte(""), 5),
				vm.NewLog(thirdAddress, []common.Hash{forthTopic}, []byte(""), 5),
				vm.NewLog(firstAddr, []common.Hash{firstTopic}, []byte(""), 5),
			}},
		}

		concatLogs = func(pl []core.PendingLogsEvent) vm.Logs {
			var logs vm.Logs
			for _, l := range pl {
				logs = append(logs, l.Logs...)
			}
			return logs
		}

		testCases = []struct {
			crit     FilterCriteria
			expected vm.Logs
			id       FilterID
		}{
			// match all
			0: {FilterCriteria{}, concatLogs(allLogs), FilterID{}},
			// match none due to no matching addresses
			1: {FilterCriteria{Addresses: []common.Address{common.Address{}, notUsedAddress}, Topics: [][]common.Hash{[]common.Hash{}}}, vm.Logs{}, FilterID{}},
			// match logs based on addresses, ignore topics
			2: {FilterCriteria{Addresses: []common.Address{firstAddr}}, append(concatLogs(allLogs[:2]), allLogs[5].Logs[3]), FilterID{}},
			// match none due to no matching topics (match with address)
			3: {FilterCriteria{Addresses: []common.Address{secondAddr}, Topics: [][]common.Hash{[]common.Hash{notUsedTopic}}}, vm.Logs{}, FilterID{}},
			// match logs based on addresses and topics
			4: {FilterCriteria{Addresses: []common.Address{thirdAddress}, Topics: [][]common.Hash{[]common.Hash{firstTopic, secondTopic}}}, append(concatLogs(allLogs[3:5]), allLogs[5].Logs[0]), FilterID{}},
			// match logs based on multiple addresses and "or" topics
			5: {FilterCriteria{Addresses: []common.Address{secondAddr, thirdAddress}, Topics: [][]common.Hash{[]common.Hash{firstTopic, secondTopic}}}, append(concatLogs(allLogs[2:5]), allLogs[5].Logs[0]), FilterID{}},
			// block numbers are ignored for filters created with New***Filter, these return all logs that match the given criterias when the state changes
			6: {FilterCriteria{Addresses: []common.Address{firstAddr}, FromBlock: rpc.BlockNumber(2), ToBlock: rpc.BlockNumber(3)}, append(concatLogs(allLogs[:2]), allLogs[5].Logs[3]), FilterID{}},
			// multiple pending logs, should match only 2 topics from the logs in block 5
			7: {FilterCriteria{Addresses: []common.Address{thirdAddress}, Topics: [][]common.Hash{[]common.Hash{firstTopic, forthTopic}}}, vm.Logs{allLogs[5].Logs[0], allLogs[5].Logs[2]}, FilterID{}},
		}

		err error
	)

	// create all filters
	for i := range testCases {
		testCases[i].id, err = filterManager.NewPendingLogFilter(testCases[i].crit, nil)
		if err != nil {
			t.Fatal(err)
		}
	}

	// raise events
	for _, l := range allLogs {
		if err := mux.Post(l); err != nil {
			t.Fatal(err)
		}
	}

	for i, tt := range testCases {
		var fetched []Log
		for {
			logs, err := filterManager.GetPendingLogFilterChanges(tt.id)
			if err != nil {
				t.Errorf("unable to retrieve logs for case %d, %s", i, err)
				break
			}
			fetched = append(fetched, logs...)

			if len(fetched) >= len(tt.expected) {
				break
			}

			time.Sleep(100 * time.Millisecond)
		}

		if len(fetched) != len(tt.expected) {
			t.Errorf("invalid number of logs for case %d, want %d log(s), got %d", i, len(tt.expected), len(fetched))
			continue
		}

		for l := range fetched {
			if fetched[l].Removed {
				t.Errorf("expected log not to be removed for log %d in case %d", l, i)
			}
			if !reflect.DeepEqual(fetched[l].Log, tt.expected[l]) {
				t.Errorf("invalid log on index %d for case %d", l, i)
			}
		}
	}
}
*/
