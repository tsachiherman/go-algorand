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
	"fmt"
	"sort"
	"time"

	"github.com/algorand/go-algorand/data/basics"
	"github.com/algorand/go-algorand/data/transactions"
	"github.com/algorand/go-algorand/protocol"
)

var _ = fmt.Printf

//msgp:ignore peerState
type peerState int

//msgp:ignore peersOps
type peersOps int

const maxIncomingBloomFilterHistory = 20
const recentTransactionsSentBufferLength = 10000
const minDataExchangeRateThreshold = 100 * 1024            // 100KB/s, which is ~0.8Mbps
const maxDataExchangeRateThreshold = 100 * 1024 * 1024 / 8 // 100Mbps
const defaultDataExchangeRateThreshold = minDataExchangeRateThreshold

const (
	// peerStateStartup is before the timeout for the sending the first message to the peer has reached.
	// for an outgoing peer, it means that an incoming message arrived, and one or more messages need to be sent out.
	peerStateStartup peerState = iota
	// peerStateHoldsoff is set once a message was sent to a peer, and we're holding off before sending additional messages.
	peerStateHoldsoff
	// peerStateInterrupt is set once the holdoff period for the peer have expired.
	peerStateInterrupt
	// peerStateLateBloom is set for outgoing peers on relays, indicating that the next message should be a bloom filter only message.
	peerStateLateBloom

	peerOpsSendMessage        peersOps = 1
	peerOpsSetInterruptible   peersOps = 2
	peerOpsClearInterruptible peersOps = 4
	peerOpsReschedule         peersOps = 8
)

// incomingBloomFilter stores an incoming bloom filter, along with the associated round number.
// the round number allow us to prune filters from rounds n-2 and below.
type incomingBloomFilter struct {
	filter bloomFilter
	round  basics.Round
}

// Peer contains peer-related data which extends the data "known" and managed by the network package.
type Peer struct {
	// networkPeer is the network package exported peer. It's created on construction and never change afterward.
	networkPeer interface{}
	// isOutgoing defines whether the peer is an outgoing peer or not. For relays, this is meaningful as these have
	// slighly different message timing logic.
	isOutgoing bool
	// state defines the peer state ( in terms of state machine state ). It's touched only by the sync main state machine
	state peerState

	lastRound basics.Round

	incomingMessages messageOrderingHeap

	nextReceivedMessageSeq uint64 // the next message seq that we expect to recieve from that peer; implies that all previous messages have been accepted.

	recentIncomingBloomFilters []incomingBloomFilter
	recentSentTransactions     *transactionLru

	// these two fields describe "what does that peer asked us to send it"
	requestedTransactionsModulator byte
	requestedTransactionsOffset    byte

	lastSentMessageSequenceNumber uint64
	lastSentMessageRound          basics.Round
	lastSentMessageTimestamp      time.Duration
	lastSentMessageSize           int
	lastSentBloomFilter           bloomFilter

	lastConfirmedMessageSeqReceived    uint64 // the last message that was confirmed by the peer to have been accepted.
	lastReceivedMessageLocalRound      basics.Round
	lastReceivedMessageTimestamp       time.Duration
	lastReceivedMessageSize            int
	lastReceivedMessageNextMsgMinDelay time.Duration

	dataExchangeRate uint64 // the combined upload/download rate in bytes/second

	// these two fields describe "what does the local peer want the remote peer to send back"
	localTransactionsModulator  byte
	localTransactionsBaseOffset byte

	// lastTransactionSelectionGroupCounter is the last transaction group counter that we've evaluated on the selectPendingTransactions method.
	// it used to ensure that on subsequent calls, we won't need to scan the entire pending transactions array from the begining.
	lastTransactionSelectionGroupCounter uint64
}

func makePeer(networkPeer interface{}, isOutgoing bool) *Peer {
	return &Peer{
		networkPeer:            networkPeer,
		isOutgoing:             isOutgoing,
		recentSentTransactions: makeTransactionLru(recentTransactionsSentBufferLength),
		dataExchangeRate:       defaultDataExchangeRateThreshold,
	}
}

// outgoing related methods :

func (p *Peer) selectPendingTransactions(pendingTransactions []transactions.SignedTxGroup, sendWindow time.Duration, round basics.Round) (selectedTxns []transactions.SignedTxGroup, selectedTxnIDs []transactions.Txid, partialTranscationsSet bool) {
	// if peer is too far back, don't send it any transactions ( or if the peer is not interested in transactions )
	if p.lastRound < round.SubSaturate(1) || p.requestedTransactionsModulator == 0 {
		return nil, nil, false
	}
	if len(pendingTransactions) == 0 {
		return nil, nil, false
	}

	windowLengthBytes := int(uint64(sendWindow) * p.dataExchangeRate / uint64(time.Second))

	accumulatedSize := 0
	selectedTxnIDs = make([]transactions.Txid, 0, len(pendingTransactions))

	startIndex := sort.Search(len(pendingTransactions), func(i int) bool {
		return pendingTransactions[i].GroupCounter >= p.lastTransactionSelectionGroupCounter
	}) % len(pendingTransactions)

	windowSizedReached := false
	hasMorePendingTransactions := false

	for scanIdx := range pendingTransactions {
		grpIdx := (scanIdx + startIndex) % len(pendingTransactions)

		// filter out transactions that we already previously sent.
		txID := pendingTransactions[grpIdx].Transactions[0].ID()
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

		// check if the peer alrady received these messages from a different source other than us.
		for filterIdx := len(p.recentIncomingBloomFilters) - 1; filterIdx >= 0; filterIdx-- {
			if p.recentIncomingBloomFilters[filterIdx].filter.test(txID) {
				continue
			}
		}

		p.lastTransactionSelectionGroupCounter = pendingTransactions[grpIdx].GroupCounter

		if windowSizedReached {
			hasMorePendingTransactions = true
			break
		}
		selectedTxns = append(selectedTxns, pendingTransactions[grpIdx])
		selectedTxnIDs = append(selectedTxnIDs, txID)

		// calculate the total size of the transaction group.
		for txidx := range pendingTransactions[grpIdx].Transactions {
			encodingBuf := protocol.GetEncodingBuf()
			accumulatedSize += len(pendingTransactions[grpIdx].Transactions[txidx].MarshalMsg(encodingBuf))
			protocol.PutEncodingBuf(encodingBuf)
		}
		if accumulatedSize > windowLengthBytes {
			windowSizedReached = true
		}
	}
	//fmt.Printf("selectPendingTransactions : selected %d transactions, and aborted after exceeding data length %d/%d more = %v\n", len(selectedTxnIDs), accumulatedSize, windowLengthBytes, hasMorePendingTransactions)

	return selectedTxns, selectedTxnIDs, hasMorePendingTransactions
}

// getLocalRequestParams returns the local requests params
func (p *Peer) getLocalRequestParams() (offset, modulator byte) {
	return p.localTransactionsBaseOffset, p.localTransactionsModulator
}

// update the peer once the message was sent successfully.
func (p *Peer) updateMessageSent(txMsg *transactionBlockMessage, selectedTxnIDs []transactions.Txid, timestamp time.Duration, sequenceNumber uint64, messageSize int, filter bloomFilter) {
	for _, txid := range selectedTxnIDs {
		p.recentSentTransactions.add(txid)
	}
	p.lastSentMessageSequenceNumber = sequenceNumber
	p.lastSentMessageRound = txMsg.Round
	p.lastSentMessageTimestamp = timestamp
	p.lastSentMessageSize = messageSize
	if filter.filter != nil {
		p.lastSentBloomFilter = filter
	}
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

// peers array functions

// incomingPeersOnly scan the input peers array and return a subset of the peers that are incoming peers.
func incomingPeersOnly(peers []*Peer) (incomingPeers []*Peer) {
	incomingPeers = make([]*Peer, 0, len(peers))
	for _, peer := range peers {
		if !peer.isOutgoing {
			incomingPeers = append(incomingPeers, peer)
		}
	}
	return
}

// incoming related functions

func (p *Peer) addIncomingBloomFilter(round basics.Round, incomingFilter bloomFilter, currentRound basics.Round) {
	bf := incomingBloomFilter{
		round:  round,
		filter: incomingFilter,
	}
	// scan the current list and find if we can removed entries.
	firstValidEntry := sort.Search(len(p.recentIncomingBloomFilters), func(i int) bool {
		return p.recentIncomingBloomFilters[i].round >= currentRound.SubSaturate(1)
	})
	if firstValidEntry < len(p.recentIncomingBloomFilters) {
		// delete some of the old entries.
		p.recentIncomingBloomFilters = p.recentIncomingBloomFilters[firstValidEntry:]
	}
	p.recentIncomingBloomFilters = append(p.recentIncomingBloomFilters, bf)
	if len(p.recentIncomingBloomFilters) > maxIncomingBloomFilterHistory {
		p.recentIncomingBloomFilters = p.recentIncomingBloomFilters[1:]
	}
}

func (p *Peer) updateRequestParams(modulator, offset byte) {
	p.requestedTransactionsModulator, p.requestedTransactionsOffset = modulator, offset
}

// update the recentSentTransactions with the incoming transaction groups. This would prevent us from sending the received transactions back to the
// peer that sent it to us.
func (p *Peer) updateIncomingTransactionGroups(txnGroups []transactions.SignedTxGroup) {
	for _, txnGroup := range txnGroups {
		if len(txnGroup.Transactions) > 0 {
			txID := txnGroup.Transactions[0].ID()
			p.recentSentTransactions.add(txID)
		}
	}
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
			//fmt.Printf("incoming message : updating data exchange to %d; network msg size = %d+%d, transmit time = %v\n", dataExchangeRate, p.lastSentMessageSize, incomingMessageSize, networkTrasmitTime)
		}
	}
	p.lastReceivedMessageLocalRound = currentRound
	p.lastReceivedMessageTimestamp = currentTime
	p.lastReceivedMessageSize = incomingMessageSize
	p.lastReceivedMessageNextMsgMinDelay = time.Duration(timings.NextMsgMinDelay) * time.Nanosecond
}

// peer state changes
func (p *Peer) advancePeerState(currenTime time.Duration, isRelay bool) (ops peersOps) {
	if isRelay {
		if p.isOutgoing {
			// outgoing peers are "special", as they respond to messages rather then generating their own.
			// we need to figure the special state needed for "late bloom filter message"
			switch p.state {
			case peerStateStartup:
				messagesCount := p.lastReceivedMessageNextMsgMinDelay / messageTimeWindow
				if messagesCount <= 1 {
					// we have time to send only a single message. This message need to include both transactions and bloom filter.
					p.state = peerStateLateBloom
				} else {
					// we have enough time to send multiple messages, make the first n-1 message have no bloom filter, and have the last one
					// include a bloom filter.
					p.state = peerStateHoldsoff
				}

				// send a message
				ops |= peerOpsSendMessage
			case peerStateHoldsoff:
				// calculate how more messages we can send ( if needed )
				messagesCount := (p.lastReceivedMessageTimestamp + p.lastReceivedMessageNextMsgMinDelay - currenTime) / messageTimeWindow
				if messagesCount <= 1 {
					// we have time to send only a single message. This message need to include both transactions and bloom filter.
					p.state = peerStateLateBloom
				}

				// send a message
				ops |= peerOpsSendMessage

				// the rescehduling would be done in the sendMessageLoop, since we need to know if additional messages are needed.
			case peerStateLateBloom:
				// send a message
				ops |= peerOpsSendMessage

			default:
				// this isn't expected, so we can just ignore this.
				// todo : log
			}
		} else {
			// non-outgoing
			switch p.state {
			case peerStateStartup:
				p.state = peerStateHoldsoff
				fallthrough
			case peerStateHoldsoff:
				// prepare the send message array.
				ops |= peerOpsSendMessage
			default: // peerStateInterrupt & peerStateLateBloom
				// this isn't expected, so we can just ignore this.
				// todo : log
			}
		}
	} else {
		switch p.state {
		case peerStateHoldsoff:
			p.state = peerStateInterrupt
			ops |= peerOpsReschedule | peerOpsSetInterruptible
		case peerStateStartup:
			fallthrough
		case peerStateInterrupt:
			p.state = peerStateHoldsoff
			ops |= peerOpsSendMessage | peerOpsClearInterruptible
		default: // peerStateLateBloom
			// this isn't expected, so we can just ignore this.
			// todo : log
		}
	}
	return
}

func (p *Peer) getNextScheduleOffset(isRelay bool, beta time.Duration, partialMessage bool) time.Duration {
	if partialMessage {
		if isRelay {
			if p.isOutgoing {
				if p.state == peerStateHoldsoff {
					// we have enough time to send another message.
					return messageTimeWindow
				}
			} else {
				return messageTimeWindow
			}
		} else {
			// update state.
			p.state = peerStateStartup
			return messageTimeWindow
		}
	} else {
		if isRelay {
			if !p.isOutgoing {
				return beta * 2
			}
		} else {
			return beta
		}
	}
	return time.Duration(0)
}
