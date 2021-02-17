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

// Package node is the Algorand node itself, with functions exposed to the frontend
package node

import (
	"context"
	"time"

	"github.com/algorand/go-algorand/data/basics"
	"github.com/algorand/go-algorand/data/transactions"
	"github.com/algorand/go-algorand/network"
	"github.com/algorand/go-algorand/protocol"
	"github.com/algorand/go-algorand/txnsync"
	"github.com/algorand/go-algorand/util/timers"
)

// transcationSyncNodeConnector implementes the txnsync.NodeConnector interface, allowing the
// transaction sync communicate with the node and it's child objects.
type transcationSyncNodeConnector struct {
	node     *AlgorandFullNode
	eventsCh chan txnsync.Event
	clock    timers.Clock
}

func makeTranscationSyncNodeConnector(node *AlgorandFullNode) transcationSyncNodeConnector {
	return transcationSyncNodeConnector{
		node:     node,
		eventsCh: make(chan txnsync.Event, 1),
		clock:    timers.MakeMonotonicClock(time.Now()),
	}
}

func (tsnc *transcationSyncNodeConnector) Events() <-chan txnsync.Event {
	return tsnc.eventsCh
}

func (tsnc *transcationSyncNodeConnector) CurrentRound() basics.Round {
	return tsnc.node.ledger.Latest()
}

func (tsnc *transcationSyncNodeConnector) Random(rng uint64) uint64 {
	return tsnc.node.Uint64() % rng
}

func (tsnc *transcationSyncNodeConnector) Clock() timers.Clock {
	return tsnc.clock
}

func (tsnc *transcationSyncNodeConnector) GetPeers() (txsyncPeers []*txnsync.Peer, netPeers []interface{}) {
	networkPeers := tsnc.node.net.GetPeers(network.PeersConnectedOut, network.PeersConnectedIn)
	txsyncPeers = make([]*txnsync.Peer, len(networkPeers))
	netPeers = make([]interface{}, len(networkPeers))
	for i := range networkPeers {
		netPeers[i] = networkPeers[i]
		txsyncPeers[i] = tsnc.node.net.GetPeerData(networkPeers[i], "txsync").(*txnsync.Peer)
	}
	return txsyncPeers, netPeers
}

func (tsnc *transcationSyncNodeConnector) UpdatePeers(txsyncPeers []*txnsync.Peer, netPeers []interface{}) {
	for i, netPeer := range netPeers {
		tsnc.node.net.SetPeerData(netPeer, "txsync", txsyncPeers[i])
	}
}

func (tsnc *transcationSyncNodeConnector) SendPeerMessage(netPeer interface{}, msg []byte) error {
	unicastPeer := netPeer.(network.UnicastPeer)
	if unicastPeer == nil {
		return nil
	}
	return unicastPeer.Unicast(context.Background(), msg, protocol.Txn2Tag)
}

func (tsnc *transcationSyncNodeConnector) GetPendingTransactionGroups() [][]transactions.SignedTxn {
	return tsnc.node.transactionPool.PendingTxGroups()
}

func (tsnc *transcationSyncNodeConnector) onNewTransactionPoolEntry(transcationPoolSize int) {
	select {
	case tsnc.eventsCh <- txnsync.MakeTranscationPoolChangeEvent(transcationPoolSize):
	default:
	}
}

func (tsnc *transcationSyncNodeConnector) onNewRound(round basics.Round, hasParticipationKeys bool) {
	// if this is a relay, then we always want to fetch transactions, regardless if we have participation keys.
	fetchTransactions := hasParticipationKeys
	if tsnc.node.config.NetAddress != "" {
		fetchTransactions = true
	}

	select {
	case tsnc.eventsCh <- txnsync.MakeNewRoundEvent(round, fetchTransactions):
	default:
	}
}
