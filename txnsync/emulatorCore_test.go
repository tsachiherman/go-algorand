// Copyright (C) 2019-2021 Algorand, Inc.
// This file is part of go-algorand
//
// go-algorand is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// go-algorand is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with go-algorand.  If not, see <https://www.gnu.org/licenses/>.

package txnsync

import (
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/algorand/go-algorand/crypto"
	"github.com/algorand/go-algorand/data/basics"
	"github.com/algorand/go-algorand/data/transactions"
	"github.com/algorand/go-algorand/logging"
	"github.com/algorand/go-algorand/protocol"
	"github.com/algorand/go-algorand/util/timers"
)

const roundDuration = 4 * time.Second

type emulator struct {
	scenario     scenario
	nodes        []*emulatedNode
	syncers      []*Service
	nodeCount    int
	log          logging.Logger
	currentRound basics.Round
	clock        timers.WallClock
}

type nodeTransaction struct {
	expirationRound basics.Round
	transactionSize int
}

type nodeTransactions []nodeTransaction

type emulatorResult struct {
	nodes []nodeTransactions
}

func (a nodeTransactions) Len() int      { return len(a) }
func (a nodeTransactions) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a nodeTransactions) Less(i, j int) bool {
	if a[i].expirationRound < a[j].expirationRound {
		return true
	}
	if a[i].expirationRound > a[j].expirationRound {
		return false
	}
	return a[i].transactionSize < a[j].transactionSize
}

func emulateScenario(t *testing.T, scenario scenario) {
	e := &emulator{
		scenario:  scenario,
		nodeCount: len(scenario.netConfig.nodes),
		log:       logging.TestingLog(t),
	}
	e.initNodes()
	e.run()

	results := e.collectResult()
	for n := range scenario.expectedResults.nodes {
		sort.Stable(scenario.expectedResults.nodes[n])
	}
	for n := range results.nodes {
		sort.Stable(results.nodes[n])
	}
	require.Equal(t, scenario.expectedResults, results)
}

func (e *emulator) run() {
	guidedClock := makeGuidedClock()
	lastRoundStarted := guidedClock.Since()
	e.clock = guidedClock
	e.start()
	// start the nodes
	for e.clock.Since() < e.scenario.testDuration {
		e.step()
		if guidedClock.Since() > lastRoundStarted+roundDuration {
			e.nextRound()
			lastRoundStarted = guidedClock.Since()
		}
		time.Sleep(1 * time.Millisecond / 10)
		guidedClock.Advance(1 * time.Millisecond)
	}
	// stop the nodes
	e.stop()
}
func (e *emulator) nextRound() {
	e.currentRound++
	for _, node := range e.nodes {
		node.onNewRound(e.currentRound, true)
	}
}
func (e *emulator) step() {
	for _, node := range e.nodes {
		node.step()
	}
}
func (e *emulator) start() {
	for _, node := range e.syncers {
		node.Start()
	}
}
func (e *emulator) stop() {
	for _, node := range e.syncers {
		node.Stop()
	}
}
func (e *emulator) initNodes() {
	e.nodes = make([]*emulatedNode, e.nodeCount, e.nodeCount)
	for i := 0; i < e.nodeCount; i++ {
		e.nodes[i] = makeEmulatedNode(e, i)
	}
	for i := 0; i < e.nodeCount; i++ {
		syncer := MakeTranscationSyncService(e.log, e.nodes[i], e.scenario.netConfig.nodes[i].isRelay)
		e.syncers = append(e.syncers, syncer)
	}
	for _, initAlloc := range e.scenario.initialAlloc {
		node := e.nodes[initAlloc.node]
		for i := 0; i < initAlloc.transactionsCount; i++ {
			var group = transactions.SignedTxGroup{}
			group.LocallyOriginated = true
			group.GroupCounter = uint64(len(node.txpoolEntries))
			group.Transactions = []transactions.SignedTxn{
				transactions.SignedTxn{
					Txn: transactions.Transaction{
						Type: protocol.PaymentTx,
						Header: transactions.Header{
							Note:      make([]byte, initAlloc.transactionSize, initAlloc.transactionSize),
							LastValid: initAlloc.expirationRound,
						},
					},
				},
			}
			crypto.RandBytes(group.Transactions[0].Txn.Note[:])
			node.txpoolIds[group.Transactions[0].ID()] = true
			node.txpoolEntries = append(node.txpoolEntries, group)
		}
		node.onNewTransactionPoolEntry()
	}
}

func (e *emulator) collectResult() (result emulatorResult) {
	result.nodes = make([]nodeTransactions, len(e.nodes))
	for i, node := range e.nodes {
		var txns nodeTransactions
		for _, txnGroup := range node.txpoolEntries {
			size := len(txnGroup.Transactions[0].Txn.Note)
			exp := txnGroup.Transactions[0].Txn.LastValid
			txns = append(txns, nodeTransaction{expirationRound: exp, transactionSize: size})
		}
		result.nodes[i] = txns
	}
	return result
}
