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
	"github.com/algorand/go-algorand/crypto"
	"github.com/algorand/go-algorand/data/basics"
	"github.com/algorand/go-algorand/data/transactions"
	"github.com/algorand/go-algorand/util/timers"
)

/*

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
*/

type networkPeer struct {
	peer          *Peer
	uploadSpeed   uint64
	downloadSpeed uint64
	isOutgoing    bool
	outSeq        uint64
	inSeq         uint64
	target        int
	messageQ      [][]byte // incoming message queue
}

// emulatedNode implements the NodeConnector interface
type emulatedNode struct {
	externalEvents chan Event
	emulator       *emulator
	peers          map[int]*networkPeer
	nodeIndex      int
	txpoolEntries  []transactions.SignedTxGroup
	txpoolIds      map[transactions.Txid]bool
}

func makeEmulatedNode(emulator *emulator, nodeIdx int) *emulatedNode {
	en := &emulatedNode{
		emulator:       emulator,
		peers:          make(map[int]*networkPeer),
		externalEvents: make(chan Event, 10000),
		nodeIndex:      nodeIdx,
		txpoolIds:      make(map[transactions.Txid]bool),
	}
	// add outgoing connections
	for _, conn := range emulator.scenario.netConfig.nodes[nodeIdx].outgoingConnections {
		en.peers[conn.target] = &networkPeer{
			uploadSpeed:   conn.uploadSpeed,
			downloadSpeed: conn.downloadSpeed,
			isOutgoing:    true,
			target:        conn.target,
		}
	}
	// add incoming connections
	for nodeID, nodeConfig := range emulator.scenario.netConfig.nodes {
		if nodeID == nodeIdx {
			continue
		}
		for _, conn := range nodeConfig.outgoingConnections {
			if conn.target != nodeIdx {
				continue
			}
			// the upload & download speeds are in reverse. This isn't a bug since we want the incoming
			// connection to be the opposite side of the connection.
			en.peers[nodeID] = &networkPeer{
				uploadSpeed:   conn.downloadSpeed,
				downloadSpeed: conn.uploadSpeed,
				isOutgoing:    false,
				target:        nodeID,
			}
		}
	}
	return en
}

func (n *emulatedNode) Events() <-chan Event {
	return n.externalEvents
}
func (n *emulatedNode) CurrentRound() basics.Round {
	return n.emulator.currentRound

}
func (n *emulatedNode) Clock() timers.WallClock {
	return n.emulator.clock.Zero().(timers.WallClock)
}

func (n *emulatedNode) Random(x uint64) (out uint64) {
	limit := x
	x += uint64(n.nodeIndex) * 997
	x += uint64(n.emulator.currentRound) * 797
	bytes := make([]byte, 8)
	for i := 0; i < 8; i++ {
		bytes[i] = byte(x >> (i * 8))
	}
	digest := crypto.Hash(bytes)
	out = 0
	for i := 0; i < 8; i++ {
		out = out << 8
		out += uint64(digest[i])
	}
	out = out % limit
	return out
}

func (n *emulatedNode) GetPeers() (out []PeerInfo) {
	for _, peer := range n.peers {
		out = append(out, PeerInfo{TxnSyncPeer: peer.peer, NetworkPeer: peer, IsOutgoing: peer.isOutgoing})
	}
	return out
}

func (n *emulatedNode) GetPeer(p interface{}) PeerInfo {
	netPeer := p.(*networkPeer)
	return PeerInfo{
		TxnSyncPeer: netPeer.peer,
		IsOutgoing:  netPeer.isOutgoing,
		NetworkPeer: p,
	}
}

func (n *emulatedNode) UpdatePeers(txPeers []*Peer, netPeers []interface{}) {
	for i, peer := range netPeers {
		netPeer := peer.(*networkPeer)
		netPeer.peer = txPeers[i]
	}
}

func (n *emulatedNode) SendPeerMessage(netPeer interface{}, msg []byte, callback SendMessageCallback) {
	peer := netPeer.(*networkPeer)
	otherNode := n.emulator.nodes[peer.target]
	otherNode.peers[n.nodeIndex].messageQ = append(otherNode.peers[n.nodeIndex].messageQ, msg)
	callback(true, peer.outSeq)
	peer.outSeq++
}

func (n *emulatedNode) GetPendingTransactionGroups() []transactions.SignedTxGroup {
	return n.txpoolEntries
}

func (n *emulatedNode) IncomingTransactionGroups(peer interface{}, groups []transactions.SignedTxGroup) {
	// add to transaction pool.
	for _, group := range groups {
		txID := group.Transactions[0].ID()
		if !n.txpoolIds[txID] {
			n.txpoolIds[txID] = true
			n.txpoolEntries = append(n.txpoolEntries, group)
		}
	}
}

func (n *emulatedNode) step() {
	msgHandler := n.emulator.syncers[n.nodeIndex].GetIncomingMessageHandler()
	// check if we have any pending network messages and forward them.
	for _, peer := range n.peers {
		for len(peer.messageQ) > 0 {
			msgHandler(peer, peer.peer, peer.messageQ[0], peer.inSeq)
			peer.inSeq++
			peer.messageQ = peer.messageQ[1:]
		}
	}

}
func (n *emulatedNode) onNewRound(round basics.Round, hasParticipationKeys bool) {
	// if this is a relay, then we always want to fetch transactions, regardless if we have participation keys.
	fetchTransactions := hasParticipationKeys
	if n.emulator.scenario.netConfig.nodes[n.nodeIndex].isRelay {
		fetchTransactions = true
	}

	n.externalEvents <- MakeNewRoundEvent(round, fetchTransactions)
}

func (n *emulatedNode) onNewTransactionPoolEntry() {
	n.externalEvents <- MakeTranscationPoolChangeEvent(len(n.txpoolEntries))
}
