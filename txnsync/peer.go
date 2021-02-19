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

//msgp:ignore peerState
type peerState int

const maxIncomingBloomFilterHistory = 20
const recentTransactionsSentBufferLength = 10000
const minDataExchangeRateThreshold = 100 * 1024        // 100KB/s, which is ~0.8Mbps
const maxDataExchangeRateThreshold = 100 * 1024 * 1024 // 100Mbps
const defaultDataExchangeRateThreshold = minDataExchangeRateThreshold

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

	// these two fields describe "what does that peer asked us to send it"
	requestedTransactionsModulator byte
	requestedTransactionsOffset    byte

	lastSentMessageSequenceNumber uint64
	lastSentMessageRound          basics.Round
	lastSentMessageTimestamp      time.Duration
	lastSentMessageSize           int

	dataExchangeRate uint64 // the combined upload/download rate in bytes/second

	// these two fields describe "what does the local peer want the remote peer to send back"
	localTransactionsModulator  byte
	localTransactionsBaseOffset byte
}

func makePeer(networkPeer interface{}) *Peer {
	return &Peer{
		networkPeer:            networkPeer,
		recentSentTransactions: makeTransactionLru(recentTransactionsSentBufferLength),
		dataExchangeRate:       defaultDataExchangeRateThreshold,
	}
}

func (p *Peer) selectPendingMessages(pendingMessages [][]transactions.SignedTxn, sendWindow time.Duration, round basics.Round) (selectedTxns [][]transactions.SignedTxn, selectedTxnIDs []transactions.Txid) {
	// if peer is too far back, don't send it any transactions ( or if the peer is not interested in transactions )
	if p.lastRound < round.SubSaturate(1) || p.requestedTransactionsModulator == 0 {
		return nil, nil
	}

	windowLengthBytes := int(uint64(sendWindow) * p.dataExchangeRate / uint64(time.Second))

	accumulatedSize := 0
	selectedTxnIDs = make([]transactions.Txid, 0, len(pendingMessages))
	for grpIdx := range pendingMessages {
		// filter out transactions that we already previously sent.
		txID := pendingMessages[grpIdx][0].ID()
		if p.recentSentTransactions.contained(txID) {
			// we already sent that transaction. no need to send again.
			continue
		}

		// check if the peer would be interested in these messages -
		if p.requestedTransactionsModulator > 1 {
			txidValue := uint64(txID[0]) + (uint64(txID[1]) << 8) + (uint64(txID[2]) << 16) + (uint64(txID[3]) << 24) + (uint64(txID[4]) << 32) + (uint64(txID[5]) << 40) + (uint64(txID[6]) << 48) + (uint64(txID[7]) << 56)
			if txidValue%uint64(p.requestedTransactionsModulator) != uint64(p.requestedTransactionsOffset) {
				continue
			}
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

func (p *Peer) updateIncomingMessageTiming(timings timingParams, currentRound basics.Round, currentTime time.Duration, incomingMessageSize int) {
	p.lastConfirmedMessageSeqReceived = timings.RefTxnBlockMsgSeq
	// if we received a message that references our privious message, see if they occured on the same round
	if p.lastConfirmedMessageSeqReceived == p.lastSentMessageSequenceNumber && p.lastSentMessageRound == currentRound {
		// if so, we migth be able to calculate the bandwidth.
		timeSinceLastMessageWasSent := currentTime - p.lastSentMessageTimestamp
		if timeSinceLastMessageWasSent > time.Duration(timings.ResponseElapsedTime) {
			networkTrasmitTime := timeSinceLastMessageWasSent - time.Duration(timings.ResponseElapsedTime)
			networkMessageSize := uint64(p.lastSentMessageSize + incomingMessageSize)
			dataExchangeRate := uint64(time.Second) * networkMessageSize / uint64(networkTrasmitTime)
			if dataExchangeRate < minDataExchangeRateThreshold {
				dataExchangeRate = minDataExchangeRateThreshold
			} else if dataExchangeRate > maxDataExchangeRateThreshold {
				dataExchangeRate = maxDataExchangeRateThreshold
			}
			// clamp data exchange rate to realistic metrics
			p.dataExchangeRate = dataExchangeRate
		}
	}
}

// update the peer once the message was sent successfully.
func (p *Peer) updateMessageSent(txMsg *transactionBlockMessage, selectedTxnIDs []transactions.Txid, timestamp time.Duration, sequenceNumber uint64, messageSize int) {
	for _, txid := range selectedTxnIDs {
		p.recentSentTransactions.add(txid)
	}
	p.lastSentMessageSequenceNumber = sequenceNumber
	p.lastSentMessageRound = txMsg.Round
	p.lastSentMessageTimestamp = timestamp
	p.lastSentMessageSize = messageSize
}

// setLocalRequestParams stores the peer request params.
func (p *Peer) setLocalRequestParams(offset, modulator uint64) {
	if modulator > 255 {
		modulator = 255
	}
	p.localTransactionsModulator = byte(modulator)
	if modulator != 0 {
		p.localTransactionsBaseOffset = byte(offset % modulator)
	}
}

// getLocalRequestParams returns the local requests params
func (p *Peer) getLocalRequestParams() (offset, modulator byte) {
	return p.localTransactionsBaseOffset, p.localTransactionsModulator
}
