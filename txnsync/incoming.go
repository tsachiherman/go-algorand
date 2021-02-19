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
	"errors"
)

var errUnsupportedTransactionSyncMessageVersion = errors.New("unsupported transaction sync message version")

type incomingMessage struct {
	networkPeer    interface{}
	message        transactionBlockMessage
	sequenceNumber uint64
	peer           *Peer
	encodedSize    int
}

// incomingMessageHandler
// note - this message is called by the network go-routine dispatch pool, and is not syncronized with the rest of the transaction syncronizer
func (s *syncState) asyncIncomingMessageHandler(networkPeer interface{}, peer *Peer, message []byte, sequenceNumber uint64) error {
	var txMsg transactionBlockMessage
	_, err := txMsg.UnmarshalMsg(message)
	if err != nil {
		// if we recieved a message that we cannot parse, disconnect.
		return err
	}
	if txMsg.version != txnBlockMessageVersion {
		// we receive a message from a version that we don't support, disconnect.
		return errUnsupportedTransactionSyncMessageVersion
	}
	if peer == nil {
		// if we don't have a peer, then we need to enqueue this task to be handled by the main loop since we want to ensure that
		// all the peer objects are created syncroniously.
		select {
		case s.incomingMessagesCh <- incomingMessage{networkPeer: networkPeer, message: txMsg, sequenceNumber: sequenceNumber, encodedSize: len(message)}:
		default:
			// todo - handle the case where we can't write to the channel.
		}
		return nil
	}
	err = peer.incomingMessages.enqueue(txMsg, sequenceNumber)
	if err != nil {
		return err
	}

	select {
	case s.incomingMessagesCh <- incomingMessage{peer: peer, encodedSize: len(message)}:
	default:
		// todo - handle the case where we can't write to the channel.
	}
	return nil
}

func (s *syncState) evaluateIncomingMessage(message incomingMessage) {
	peer := message.peer
	if peer == nil {
		// check if a peer was created already for this network peer object.
		peer = s.node.GetPeer(message.networkPeer)
		if peer == nil {
			// we couldn't really do much about this message previously, since we didn't have the peer.
			peer = makePeer(message.networkPeer)
			// let the network peer object know about our peer
			s.node.UpdatePeers([]*Peer{peer}, []interface{}{message.networkPeer})
		}
		err := peer.incomingMessages.enqueue(message.message, message.sequenceNumber)
		if err != nil {
			// this is not really likely, since we won't saturate the peer heap right after creating it..
			return
		}
	}
	seq, err := peer.incomingMessages.peekSequence()
	if err != nil {
		// this is not really likely, since we just enqueued an entry and all the "dequeuing" is done on this go-routine.
		return
	}
	if seq != peer.nextReceivedMessageSeq {
		// if we recieve a message which wasn't in-order, just let it go.
		return
	}
	txMsg, err := peer.incomingMessages.pop()
	if err != nil {
		// if the queue is empty ( not expected, since we peek'ed into it before ), then we can't do much here.
		return
	}

	// increase the message sequence number, since we're processing this message.
	peer.nextReceivedMessageSeq++

	// update the round number if needed.
	if txMsg.round > peer.lastRound {
		peer.lastRound = txMsg.round
	}

	// if the peer sent us a bloom filter, store this.
	if txMsg.txnBloomFilter.bloomFilterType != 0 {
		bloomFilter, err := decodeBloomFilter(txMsg.txnBloomFilter)
		if err == nil {
			peer.addIncomingBloomFilter(bloomFilter)
		}
	}
	peer.updateRequestParams(txMsg.updatedRequestParams.modulator, txMsg.updatedRequestParams.offset)
	peer.updateIncomingMessageTiming(txMsg.msgSync, s.round, s.node.Clock().Since(), message.encodedSize)

	// if the peer's round is more than a single round behind the local node, then we don't want to
	// try and load the transactions. The other peer should first catch up before getting transactions.
	if (peer.lastRound + 1) < s.round {
		return
	}
	s.node.IncomingTransactionGroups(peer.networkPeer, txMsg.transactionGroups.transactionsGroup)
}