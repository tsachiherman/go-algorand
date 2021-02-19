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
	"time"

	"github.com/algorand/go-algorand/data/basics"
	"github.com/algorand/go-algorand/data/transactions"
	"github.com/algorand/go-algorand/protocol"
)

type peerState int

const (
	// peerStateStartup is before the timeout for the sending the first message to the peer has reached
	peerStateStartup peerState = iota
	// peerStateHoldsoff is set once a message was sent to a peer, and we're holding off before sending additional messages.
	peerStateHoldsoff
	// peerStateInterrupt is set once the holdoff period for the peer have expired.
	peerStateInterrupt
)

// Peer contains peer-related data which extends the data "known" and managed by the network package.
type Peer struct {
	// networkPeer is the network package exported peer. It's created on construction and never change afterward.
	networkPeer interface{}
	// state defines the peer state ( in terms of state machine state ). It's touched only by the sync main state machine
	state peerState

	lastRound basics.Round

	incomingMessages messageOrderingHeap
	nextMessageSeq   uint64
}

func makePeer(networkPeer interface{}) *Peer {
	return &Peer{
		networkPeer: networkPeer,
	}
}

// GetUploadRate return the peer upload rate in bytes/sec, based on recent recieved packets.
func (p *Peer) getUploadRate() int64 {
	return 2 * 1024 * 1024 / 8 // dummy default of 2Mbit/sec.
}

func (p *Peer) selectPendingMessages(pendingMessages [][]transactions.SignedTxn, sendWindow time.Duration) (selectedTxns [][]transactions.SignedTxn) {
	windowLengthBytes := int(int64(sendWindow) * p.getUploadRate() / int64(time.Second))

	accumulatedSize := 0
	for grpIdx := range pendingMessages {
		// todo - filter out transactions that we already previously sent.
		selectedTxns = append(selectedTxns, pendingMessages[grpIdx])

		for txidx := range pendingMessages[grpIdx] {
			encodingBuf := protocol.GetEncodingBuf()
			accumulatedSize += len(pendingMessages[grpIdx][txidx].MarshalMsg(encodingBuf))
			protocol.PutEncodingBuf(encodingBuf)
		}
		if accumulatedSize > windowLengthBytes {
			break
		}
	}

	return selectedTxns
}
