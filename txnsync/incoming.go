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
}

// incomingMessageHandler
// note - this message is called by the network go-routine dispatch pool, and is not syncronized with the rest of the transaction syncronizer
func (s *syncState) incomingMessageHandler(networkPeer interface{}, peer *Peer, message []byte, sequenceNumber uint64) error {
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
		case s.incomingMessagesCh <- incomingMessage{networkPeer: networkPeer, message: txMsg, sequenceNumber: sequenceNumber}:
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
	case s.incomingMessagesCh <- incomingMessage{peer: peer}:
	default:
		// todo - handle the case where we can't write to the channel.
	}
	return nil
}

func (s *syncState) evaluateIncomingMessages(message incomingMessage) {
	peer := message.peer
	if peer == nil {
		// we couldn't really do much about this message previously, since we didn't have the peer.
		peer := makePeer(message.networkPeer)
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
	if seq != peer.nextMessageSeq {
		// if we recieve a message which wasn't in-order, just let it go.
		return
	}
	txMsg, err := peer.incomingMessages.pop()
	if err != nil {
		// if the queue is empty ( not expected, since we peek'ed into it before ), then we can't do much here.
		return
	}
	if txMsg.round > peer.lastRound {
		peer.lastRound = txMsg.round
	}
	// todo..
}
