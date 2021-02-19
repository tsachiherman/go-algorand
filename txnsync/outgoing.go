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
	"context"
	"time"

	"github.com/algorand/go-algorand/data/transactions"
)

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
	msc.sentTimestamp = msc.state.node.Clock().Since()
	msc.sequenceNumber = sequenceNumber

	select {
	case msc.state.outgoingMessagesCallbackCh <- msc:
	default:
		// if we can't place it on the channel, just let it drop and log it.
	}
}

func (s *syncState) sendMessageLoop(ctx context.Context, peers []*Peer) {
	if len(peers) == 0 {
		// no peers - no messages that need to be sent.
		return
	}
	pendingTransactionGroups := s.node.GetPendingTransactionGroups()
	var encodedMessage []byte
	for _, peer := range peers {
		msgCallback := &messageSentCallback{peer: peer}
		encodedMessage, msgCallback.sentMessage, msgCallback.sentTranscationsIDs = s.assemblePeerMessage(peer, pendingTransactionGroups)
		s.node.SendPeerMessage(peer.networkPeer, encodedMessage, msgCallback.asyncMessageSent)

		if ctx.Err() != nil {
			// we ran out of time sending messages, stop sending any more messages.
			break
		}
	}
}

func (s *syncState) assemblePeerMessage(peer *Peer, pendingMessages [][]transactions.SignedTxn) (encodedMessage []byte, txMsg *transactionBlockMessage, sentTxIDs []transactions.Txid) {
	txMsg = &transactionBlockMessage{
		version: txnBlockMessageVersion,
		round:   s.round,
	}

	if s.fetchTransactions {
		// todo - fill TxnBloomFilter
		// todo - fill UpdatedRequestParams
	}

	//windowDuration = 20 * time.Millisecond
	txMsg.transactionGroups.transactionsGroup, sentTxIDs = peer.selectPendingMessages(pendingMessages, 20*time.Millisecond, s.round)

	return txMsg.MarshalMsg([]byte{}), txMsg, sentTxIDs
}

func (s *syncState) evaluateOutgoingMessage(msg *messageSentCallback) {
	msg.peer.updateMessageSent(msg.sentMessage, msg.sentTranscationsIDs, msg.sentTimestamp, msg.sequenceNumber, msg.encodedMessageSize)
}
