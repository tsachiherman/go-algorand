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
	"github.com/algorand/go-algorand/util/timers"
	"time"

	"github.com/algorand/go-algorand/data/transactions"
)

var _ = fmt.Printf

const messageTimeWindow = 20 * time.Millisecond

var outgoingTxSyncMsgFormat = "Outgoing Txsync #%d round %d transacations %d request [%d/%d] bloom %d"

type sentMessageMetadata struct {
	encodedMessageSize  int
	sentTranscationsIDs []transactions.Txid
	message             *transactionBlockMessage
	peer                *Peer
	sentTimestamp       time.Duration
	sequenceNumber      uint64
	partialMessage      bool
	filter              bloomFilter
}

type messageSentCallback struct {
	state       *syncState
	messageData sentMessageMetadata
}

// asyncMessageSent called via the network package to inform the txsync that a message was enqueued, and the associated sequence number.
func (msc *messageSentCallback) asyncMessageSent(enqueued bool, sequenceNumber uint64) {
	if !enqueued {
		return
	}
	// record the timestamp here, before placing the entry on the queue
	msc.messageData.sentTimestamp = msc.state.clock.Since()
	msc.messageData.sequenceNumber = sequenceNumber

	select {
	case msc.state.outgoingMessagesCallbackCh <- msc:
	default:
		// if we can't place it on the channel, just let it drop and log it.
	}
}

func (s *syncState) sendMessageLoop(currentTime time.Duration, deadline timers.DeadlineMonitor, peers []*Peer) {
	if len(peers) == 0 {
		// no peers - no messages that need to be sent.
		return
	}
	pendingTransactionGroups := s.node.GetPendingTransactionGroups()

	for _, peer := range peers {
		msgCallback := &messageSentCallback{state: s}
		msgCallback.messageData = s.assemblePeerMessage(peer, pendingTransactionGroups, currentTime)
		encodedMessage := msgCallback.messageData.message.MarshalMsg([]byte{})
		msgCallback.messageData.encodedMessageSize = len(encodedMessage)
		s.node.SendPeerMessage(peer.networkPeer, encodedMessage, msgCallback.asyncMessageSent)

		scheduleOffset := peer.getNextScheduleOffset(s.isRelay, s.lastBeta, msgCallback.messageData.partialMessage)
		if scheduleOffset > time.Duration(0) {
			s.scheduler.schedulerPeer(peer, currentTime+scheduleOffset)
		}

		if deadline.Expired() {
			// we ran out of time sending messages, stop sending any more messages.
			break
		}
	}
}

func (s *syncState) assemblePeerMessage(peer *Peer, pendingTransactions []transactions.SignedTxGroup, currentTime time.Duration) (metaMessage sentMessageMetadata) {
	metaMessage = sentMessageMetadata{
		peer: peer,
		message: &transactionBlockMessage{
			Version: txnBlockMessageVersion,
			Round:   s.round,
		},
	}

	createBloomFilter := false
	sendTransactions := false

	// on outgoing peers of relays, we want have some custom logic.
	if s.isRelay && peer.isOutgoing {
		switch peer.state {
		case peerStateStartup:
			// we need to send just the bloom filter.
			createBloomFilter = true
			//fmt.Printf("assembling message on outgoing relay : sending bloom, no tx\n")
		case peerStateLateBloom:
			sendTransactions = true
			createBloomFilter = true
			//fmt.Printf("assembling message on outgoing relay : sending tx & bloom\n")
		case peerStateHoldsoff:
			sendTransactions = true
			//fmt.Printf("assembling message on outgoing relay : sending tx, no bloom\n")
		default:
			// todo - log
		}
	} else {
		createBloomFilter = s.fetchTransactions
		sendTransactions = true
	}

	if s.fetchTransactions {
		// update the UpdatedRequestParams
		offset, modulator := peer.getLocalRequestParams()
		metaMessage.message.UpdatedRequestParams.Modulator = modulator
		if modulator > 0 {
			metaMessage.message.UpdatedRequestParams.Offset = byte((s.requestsOffset + uint64(offset)) % uint64(modulator))
		}
	}

	if createBloomFilter && len(pendingTransactions) > 0 {
		// generate a bloom filter that matches the requests params.
		metaMessage.filter = makeBloomFilter(metaMessage.message.UpdatedRequestParams, pendingTransactions, uint32(s.node.Random(0xffffffff)))
		if !metaMessage.filter.compare(peer.lastSentBloomFilter) {
			metaMessage.message.TxnBloomFilter = metaMessage.filter.encode()
		}
	}

	if sendTransactions {
		if !s.isRelay {
			// on non-relay, we need to filter out the non-locally originated messages since we don't want
			// non-relays to send transcation that they received via the transaction sync back.
			pendingTransactions = locallyGeneratedTransactions(pendingTransactions)
		}
		var txnGroups []transactions.SignedTxGroup
		txnGroups, metaMessage.sentTranscationsIDs, metaMessage.partialMessage = peer.selectPendingTransactions(pendingTransactions, messageTimeWindow, s.round)
		metaMessage.message.TransactionGroups.Bytes = encodeTransactionGroups(txnGroups)
	}
	/*if len(txnGroups) > 0 {
		fmt.Printf("sent transactions groups %d (%d bytes)\n", len(txnGroups), len(txMsg.TransactionGroups.Bytes))
	}*/

	metaMessage.message.MsgSync.RefTxnBlockMsgSeq = peer.nextReceivedMessageSeq - 1
	if peer.lastReceivedMessageTimestamp != 0 && peer.lastReceivedMessageLocalRound == s.round {
		metaMessage.message.MsgSync.ResponseElapsedTime = uint64((currentTime - peer.lastReceivedMessageTimestamp).Nanoseconds())
	}

	if s.isRelay {
		if peer.isOutgoing {
			metaMessage.message.MsgSync.NextMsgMinDelay = uint64(s.lastBeta.Nanoseconds()) // todo - find a better way to caluclate this.
		} else {
			metaMessage.message.MsgSync.NextMsgMinDelay = uint64(s.lastBeta.Nanoseconds()) * 2
		}
	} else {
		metaMessage.message.MsgSync.NextMsgMinDelay = uint64(s.lastBeta.Nanoseconds())
	}
	return metaMessage
}

func (s *syncState) evaluateOutgoingMessage(msg *messageSentCallback) {
	msgData := msg.messageData
	msgData.peer.updateMessageSent(msgData.message, msgData.sentTranscationsIDs, msgData.sentTimestamp, msgData.sequenceNumber, msgData.encodedMessageSize, msgData.filter)
	s.log.Infof(outgoingTxSyncMsgFormat, msgData.sequenceNumber, msgData.message.Round, len(msgData.sentTranscationsIDs), msgData.message.UpdatedRequestParams.Offset, msgData.message.UpdatedRequestParams.Modulator, len(msgData.message.TxnBloomFilter.BloomFilter))
}

// locallyGeneratedTransactions return a subset of the given transactionGroups array by filtering out transactions that are not locally generated.
func locallyGeneratedTransactions(transactionGroups []transactions.SignedTxGroup) (result []transactions.SignedTxGroup) {
	result = make([]transactions.SignedTxGroup, 0, len(transactionGroups))
	for _, txnGroup := range transactionGroups {
		if txnGroup.LocallyOriginated {
			result = append(result, txnGroup)
		}
	}
	return transactionGroups
}
