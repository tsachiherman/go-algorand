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
	"math"
	"sync"
	"time"

	"github.com/algorand/go-algorand/data/basics"
	"github.com/algorand/go-algorand/logging"
)

const (
	kickoffTime      = 200 * time.Millisecond
	randomRange      = 100 * time.Millisecond
	sendMessagesTime = 10 * time.Millisecond
)

type syncState struct {
	service *Service
	log     logging.Logger
	node    NodeConnector

	lastBeta           time.Duration
	round              basics.Round
	fetchTransactions  bool
	scheduler          peerScheduler
	interruptablePeers map[*Peer]bool
}

func (s *syncState) mainloop(serviceCtx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	s.interruptablePeers = make(map[*Peer]bool)
	s.scheduler.node = s.node
	s.lastBeta = s.beta(0)
	startRound := s.node.CurrentRound()
	s.onNewRoundEvent(MakeNewRoundEvent(startRound, false))

	externalEvents := s.node.Events()
	clock := s.node.Clock()
	var nextSyncCh <-chan time.Time
	for {
		nextAction := s.scheduler.nextDuration()
		if nextAction != time.Duration(0) {
			nextSyncCh = clock.TimeoutAt(nextAction)
		} else {
			nextSyncCh = nil
		}
		select {
		case ent := <-externalEvents:
			switch ent.eventType {
			case transactionPoolChangedEvent:
				s.onTransactionPoolChangedEvent(ent)
			case newRoundEvent:
				s.onNewRoundEvent(ent)
			}
		case <-nextSyncCh:
			s.evaluateSendingMessages(nextAction)
			s.log.Infof("sync time")
		case <-serviceCtx.Done():
			return
		}
	}
}

func (s *syncState) onTransactionPoolChangedEvent(ent Event) {
	newBeta := s.beta(ent.transactionPoolSize)
	// see if the newBeta is at least 20% smaller than the current one.
	if (s.lastBeta * 8 / 10) <= newBeta {
		// no, it's not.
		return
	}
	// yes, the number of transactions in the pool have changed dramatically since the last time.
	s.lastBeta = newBeta

	currentTimeout := time.Duration(0) // todo : figure this out.

	peers := make([]*Peer, 0, len(s.interruptablePeers))
	for peer := range s.interruptablePeers {
		peers = append(peers, peer)
		peer.state = peerStateHoldsoff
		s.scheduler.reschedulerPeer(peer, currentTimeout+s.lastBeta)
	}

	sendMessagesTimeout := s.node.Clock().TimeoutAt(currentTimeout + sendMessagesTime)
	s.sendMessageLoop(peers, sendMessagesTimeout)

}

// calculate the beta parameter, based on the transcation pool size.
func (s *syncState) beta(txPoolSize int) time.Duration {
	if txPoolSize < 200 {
		txPoolSize = 200
	} else if txPoolSize > 10000 {
		txPoolSize = 10000
	}
	beta := 2 * 3.6923 * math.Exp(float64(txPoolSize)*0.0003)
	return time.Duration(float64(time.Millisecond) * beta)

}

func (s *syncState) onNewRoundEvent(ent Event) {
	s.node.Clock().Zero()
	s.scheduler.scheduleNewRound()
	s.round = ent.round
	s.fetchTransactions = ent.fetchTransactions
}

func (s *syncState) evaluateSendingMessages(currentTimeout time.Duration) {
	peers := s.scheduler.nextPeers()
	if len(peers) == 0 {
		return
	}

	sendMessagePeers := 0
	for _, peer := range peers {
		switch peer.state {
		case peerStateHoldsoff:
			peer.state = peerStateInterrupt
			s.scheduler.schedulerPeer(peer, currentTimeout+s.lastBeta)
			s.interruptablePeers[peer] = true
		default: // peerStateStartup && peerStateInterrupt
			peer.state = peerStateHoldsoff
			s.scheduler.schedulerPeer(peer, currentTimeout+s.lastBeta)
			// prepare the send message array.
			peers[sendMessagePeers] = peer
			sendMessagePeers++
			delete(s.interruptablePeers, peer)
		}
	}

	peers = peers[:sendMessagePeers]
	sendMessagesTimeout := s.node.Clock().TimeoutAt(currentTimeout + sendMessagesTime)
	s.sendMessageLoop(peers, sendMessagesTimeout)
}

func (s *syncState) holdsoffDuration() time.Duration {
	return s.lastBeta
}
