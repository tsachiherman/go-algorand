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

const maxIncomingBloomFilterHistory = 20
const recentTransactionsSentBufferLength = 10000

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

	nextReceivedMessageSeq          uint64
	lastConfirmedMessageSeqReceived uint64

	recentIncomingBloomFilters []bloomFilter
	recentSentTransactions     *transactionLru

	// these two fields describe "what the other peer want us to send it"
	requestedTransactionsModulator byte
	requestedTransactionsOffset    byte
}

func makePeer(networkPeer interface{}) *Peer {
	return &Peer{
		networkPeer:            networkPeer,
		recentSentTransactions: makeTransactionLru(recentTransactionsSentBufferLength),
	}
}

// GetUploadRate return the peer upload rate in bytes/sec, based on recent recieved packets.
func (p *Peer) getUploadRate() int64 {
	return 2 * 1024 * 1024 / 8 // dummy default of 2Mbit/sec.
}

func (p *Peer) selectPendingMessages(pendingMessages [][]transactions.SignedTxn, sendWindow time.Duration) (selectedTxns [][]transactions.SignedTxn, selectedTxnIDs []transactions.Txid) {
	windowLengthBytes := int(int64(sendWindow) * p.getUploadRate() / int64(time.Second))

	accumulatedSize := 0
	selectedTxnIDs = make([]transactions.Txid, 0, len(pendingMessages))
	for grpIdx := range pendingMessages {
		// todo - filter out transactions that we already previously sent.
		txID := pendingMessages[grpIdx][0].ID()
		if p.recentSentTransactions.contained(txID) {
			// we already sent that transaction. no need to send again.
			continue
		}

		selectedTxns = append(selectedTxns, pendingMessages[grpIdx])
		selectedTxnIDs = append(selectedTxnIDs, txID)

		// calculate the total size of the transaction group.
		for txidx := range pendingMessages[grpIdx] {
			encodingBuf := protocol.GetEncodingBuf()
			accumulatedSize += len(pendingMessages[grpIdx][txidx].MarshalMsg(encodingBuf))
			protocol.PutEncodingBuf(encodingBuf)
		}
		if accumulatedSize > windowLengthBytes {
			break
		}
	}

	return selectedTxns, selectedTxnIDs
}

func (p *Peer) addIncomingBloomFilter(bf bloomFilter) {
	p.recentIncomingBloomFilters = append(p.recentIncomingBloomFilters, bf)
	if len(p.recentIncomingBloomFilters) > maxIncomingBloomFilterHistory {
		p.recentIncomingBloomFilters = p.recentIncomingBloomFilters[1:]
	}
}

func (p *Peer) updateRequestParams(modulator, offset byte) {
	p.requestedTransactionsModulator, p.requestedTransactionsOffset = modulator, offset
}

func (p *Peer) updateIncomingMessageTiming(timings timingParams) {
	p.lastConfirmedMessageSeqReceived = uint64(timings.refTxnBlockMsgSeq)
}

// update the peer once the message was sent successfully.
func (p *Peer) updateMessageSent(txMsg *transactionBlockMessage, selectedTxnIDs []transactions.Txid) {
	for _, txid := range selectedTxnIDs {
		p.recentSentTransactions.add(txid)
	}
}
