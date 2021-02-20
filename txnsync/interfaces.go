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
	"github.com/algorand/go-algorand/data/basics"
	"github.com/algorand/go-algorand/data/transactions"
	"github.com/algorand/go-algorand/util/timers"
)

//msgp:ignore eventType
type eventType int

const (
	transactionPoolChangedEvent eventType = 1
	newRoundEvent               eventType = 2
)

// Event is an external triggering event
type Event struct {
	eventType

	round               basics.Round
	transactionPoolSize int
	fetchTransactions   bool // for non-relays that has no participation keys, there is no need to request transactions
}

// IncomingMessageHandler is the signature of the incoming message handler used by the transaction sync to receive network messages
type IncomingMessageHandler func(networkPeer interface{}, peer *Peer, message []byte, sequenceNumber uint64) error

// SendMessageCallback define a message sent feedback for performing message tracking
type SendMessageCallback func(enqueued bool, sequenceNumber uint64)

// PeerInfo describes a single peer returned by GetPeers or GetPeer
type PeerInfo struct {
	TxnSyncPeer *Peer
	NetworkPeer interface{}
	IsOutgoing  bool
}

// NodeConnector is used by the transaction sync for communicating with components external to the txnsync package.
type NodeConnector interface {
	Events() <-chan Event
	CurrentRound() basics.Round // return the current round from the ledger.
	Clock() timers.WallClock
	Random(uint64) uint64
	GetPeers() []PeerInfo
	GetPeer(interface{}) PeerInfo // get a single peer given a network peer opaque interface
	UpdatePeers([]*Peer, []interface{})
	SendPeerMessage(netPeer interface{}, msg []byte, callback SendMessageCallback)
	GetPendingTransactionGroups() []transactions.SignedTxGroup
	IncomingTransactionGroups(interface{}, []transactions.SignedTxGroup)
}

// MakeTranscationPoolChangeEvent creates an event for when a txn pool size has changed.
func MakeTranscationPoolChangeEvent(transactionPoolSize int) Event {
	return Event{
		eventType:           transactionPoolChangedEvent,
		transactionPoolSize: transactionPoolSize,
	}
}

// MakeNewRoundEvent creates an event for when a new round starts
func MakeNewRoundEvent(roundNumber basics.Round, fetchTransactions bool) Event {
	return Event{
		eventType:         newRoundEvent,
		round:             roundNumber,
		fetchTransactions: fetchTransactions,
	}
}
