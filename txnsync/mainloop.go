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

	nextSyncTime      time.Duration
	nextInterruptTime time.Duration
	preKickoff        bool
	lastBeta          time.Duration
	interruptEnable   bool
	round             basics.Round
	fetchTransactions bool
}

func (s *syncState) mainloop(serviceCtx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	s.lastBeta = s.beta(0)
	startRound := s.node.CurrentRound()
	s.onNewRoundEvent(MakeNewRoundEvent(startRound, false))

	externalEvents := s.node.Events()
	clock := s.node.Clock()

	for {
		nextSyncCh := clock.TimeoutAt(s.nextSyncTime)
		nextInterruptCh := clock.TimeoutAt(s.nextInterruptTime)
		select {
		case ent := <-externalEvents:
			switch ent.eventType {
			case transactionPoolChangedEvent:
				s.onTransactionPoolChangedEvent(ent)
			case newRoundEvent:
				s.onNewRoundEvent(ent)
			}
		case <-nextSyncCh:
			s.sendMessages()
			s.log.Infof("sync time")
		case <-nextInterruptCh:
			s.onNextInterrupt()
		case <-serviceCtx.Done():
			return
		}
	}
}

func (s *syncState) onTransactionPoolChangedEvent(ent Event) {
	if !s.interruptEnable {
		return
	}
	newBeta := s.beta(ent.transactionPoolSize)
	// see if the newBeta is at least 20% smaller than the current one.
	if (s.lastBeta * 8 / 10) <= newBeta {
		// no, it's not.
		return
	}
	// yes, the number of transactions in the pool have changed dramatically since the last time.
	s.lastBeta = newBeta

	// reseting the clock and setting the next sync to zero would make it trigger right away.
	s.node.Clock().Zero()

	s.nextSyncTime = 0
	s.nextInterruptTime = s.holdsoffDuration()
	s.interruptEnable = false
}

func (s *syncState) onNextInterrupt() {
	// enable msg sending interrupts.
	s.interruptEnable = true
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
	s.nextSyncTime = kickoffTime + time.Duration(s.node.Random(uint64(randomRange)))
	s.nextInterruptTime = s.nextSyncTime + s.holdsoffDuration()
	s.preKickoff = true
	s.interruptEnable = false
	s.round = ent.round
	s.fetchTransactions = ent.fetchTransactions
}

func (s *syncState) sendMessages() {
	sendMessageTimeout := s.node.Clock().TimeoutAt(s.nextSyncTime + sendMessagesTime)
	s.sendMessageLoop(sendMessageTimeout)

	// update the next time a message need to be sent.
	s.nextInterruptTime = s.nextSyncTime + s.holdsoffDuration()
	s.nextSyncTime = s.nextInterruptTime + s.holdsoffDuration()
	s.interruptEnable = false
}

func (s *syncState) holdsoffDuration() time.Duration {
	return s.lastBeta
}
