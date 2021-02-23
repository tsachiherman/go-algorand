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

var outgoingTxSyncMsgFormat = "Outgoing Txsync #%d round %d transacations %d request [%d/%d]"

type messageSentCallback struct {
	encodedMessageSize  int
	sentTranscationsIDs []transactions.Txid
	sentMessage         *transactionBlockMessage
	peer                *Peer
	state               *syncState
	sentTimestamp       time.Duration
	sequenceNumber      uint64
}

// asyncMessageSent called via the network package to inform the txsync that a message was enqueued, and the associated sequence number.
func (msc *messageSentCallback) asyncMessageSent(enqueued bool, sequenceNumber uint64) {
	if !enqueued {
		return
	}
	// record the timestamp here, before placing the entry on the queue
	msc.sentTimestamp = msc.state.clock.Since()
	msc.sequenceNumber = sequenceNumber

	select {
	case msc.state.outgoingMessagesCallbackCh <- msc:
	default:
		// if we can't place it on the channel, just let it drop and log it.
	}
}

func (s *syncState) sendMessageLoop(deadline timers.DeadlineMonitor, peers []*Peer) {
	if len(peers) == 0 {
		// no peers - no messages that need to be sent.
		return
	}
	pendingTransactionGroups := s.node.GetPendingTransactionGroups()
	currentTime := s.clock.Since()
	var encodedMessage []byte
	for _, peer := range peers {
		msgCallback := &messageSentCallback{peer: peer, state: s}
		encodedMessage, msgCallback.sentMessage, msgCallback.sentTranscationsIDs = s.assemblePeerMessage(peer, pendingTransactionGroups, currentTime)
		s.node.SendPeerMessage(peer.networkPeer, encodedMessage, msgCallback.asyncMessageSent)
		if deadline.Expired() {
			// we ran out of time sending messages, stop sending any more messages.
			break
		}
	}
}

func (s *syncState) assemblePeerMessage(peer *Peer, pendingTransactions []transactions.SignedTxGroup, currentTime time.Duration) (encodedMessage []byte, txMsg *transactionBlockMessage, sentTxIDs []transactions.Txid) {
	txMsg = &transactionBlockMessage{
		Version: txnBlockMessageVersion,
		Round:   s.round,
	}

	createBloomFilter := false
	sendTransactions := false

	// on outgoing peers of relays, we want have some custom logic.
	if s.isRelay && peer.isOutgoing {
		switch peer.state {
		case peerStateStartup:
			// we need to send just the bloom filter.
			createBloomFilter = true
		case peerStateLateBloom:
			sendTransactions = true
		default:
			// todo - log
		}
	} else {
		createBloomFilter = true
		sendTransactions = true
	}

	if s.fetchTransactions {
		// update the UpdatedRequestParams
		offset, modulator := peer.getLocalRequestParams()
		txMsg.UpdatedRequestParams.Modulator = modulator
		if modulator > 0 {
			txMsg.UpdatedRequestParams.Offset = byte((s.requestsOffset + uint64(offset)) % uint64(modulator))
		}
		createBloomFilter = true
	}

	if createBloomFilter {
		// generate a bloom filter that matches the requests params.
		if len(pendingTransactions) > 0 {
			bloomFilter := makeBloomFilter(txMsg.UpdatedRequestParams, pendingTransactions, uint32(s.node.Random(0xffffffff)))
			txMsg.TxnBloomFilter = bloomFilter.encode()
		}
	}
	if sendTransactions {
		if !s.isRelay {
			// on non-relay, we need to filter out the non-locally originated messages since we don't want
			// non-relays to send transcation that they received via the transaction sync back.
			pendingTransactions = locallyGeneratedTransactions(pendingTransactions)
		}
		var txnGroups []transactions.SignedTxGroup
		txnGroups, sentTxIDs = peer.selectPendingTransactions(pendingTransactions, messageTimeWindow, s.round)
		txMsg.TransactionGroups.Bytes = encodeTransactionGroups(txnGroups)
	}
	/*if len(txnGroups) > 0 {
		fmt.Printf("sent transactions groups %d (%d bytes)\n", len(txnGroups), len(txMsg.TransactionGroups.Bytes))
	}*/

	txMsg.MsgSync.RefTxnBlockMsgSeq = peer.nextReceivedMessageSeq - 1
	if peer.lastReceivedMessageTimestamp != 0 && peer.lastReceivedMessageLocalRound == s.round {
		txMsg.MsgSync.ResponseElapsedTime = uint64((currentTime - peer.lastReceivedMessageTimestamp).Nanoseconds())
	}

	if s.isRelay {
		if peer.isOutgoing {
			txMsg.MsgSync.NextMsgMinDelay = uint64(s.lastBeta.Nanoseconds()) // todo - find a better way to caluclate this.
		} else {
			txMsg.MsgSync.NextMsgMinDelay = uint64(s.lastBeta.Nanoseconds()) * 2
		}
	} else {
		txMsg.MsgSync.NextMsgMinDelay = uint64(s.lastBeta.Nanoseconds())
	}
	return txMsg.MarshalMsg([]byte{}), txMsg, sentTxIDs
}

func (s *syncState) evaluateOutgoingMessage(msg *messageSentCallback) {
	msg.peer.updateMessageSent(msg.sentMessage, msg.sentTranscationsIDs, msg.sentTimestamp, msg.sequenceNumber, msg.encodedMessageSize)
	s.log.Infof(outgoingTxSyncMsgFormat, msg.sequenceNumber, msg.sentMessage.Round, len(msg.sentTranscationsIDs), msg.sentMessage.UpdatedRequestParams.Offset, msg.sentMessage.UpdatedRequestParams.Modulator)
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
