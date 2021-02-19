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

func (s *syncState) sendMessageLoop(ctx context.Context, peers []*Peer) {
	if len(peers) == 0 {
		// no peers - no messages that need to be sent.
		return
	}
	pendingTransactionGroups := s.node.GetPendingTransactionGroups()

	for _, peer := range peers {
		encodedMsg, txMsg, TxnIDs := s.assemblePeerMessage(peer, pendingTransactionGroups)
		err := s.node.SendPeerMessage(peer.networkPeer, encodedMsg)
		if err == nil {
			// if we successfully sent the message, we should make note of that on the peer.
			peer.updateMessageSent(txMsg, TxnIDs)

			//s.logPeerMessageSent(peer, peerMsg)
		}

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

	//peerUploadRate := peer.GetUploadRate()
	//windowDuration = 20 * time.Millisecond
	//windowMessageLength := windowDuration * peerUploadRate / time.Second
	txMsg.transactionGroups.transactionsGroup, sentTxIDs = peer.selectPendingMessages(pendingMessages, 20*time.Millisecond)
	//txMsg.msgSync.

	return txMsg.MarshalMsg([]byte{}), txMsg, sentTxIDs
}
