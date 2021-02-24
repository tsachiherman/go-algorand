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
	"github.com/algorand/go-algorand/util/timers"
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
	isRelay bool
	clock   timers.WallClock

	lastBeta                   time.Duration
	round                      basics.Round
	fetchTransactions          bool
	scheduler                  peerScheduler
	interruptablePeers         map[*Peer]bool
	incomingMessagesCh         chan incomingMessage
	outgoingMessagesCallbackCh chan *messageSentCallback
	nextOffsetRollingCh        <-chan time.Time
	requestsOffset             uint64
}

func (s *syncState) mainloop(serviceCtx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	s.clock = s.node.Clock()
	s.interruptablePeers = make(map[*Peer]bool)
	s.incomingMessagesCh = make(chan incomingMessage, 1024)
	s.outgoingMessagesCallbackCh = make(chan *messageSentCallback, 1024)
	s.scheduler.node = s.node
	s.lastBeta = beta(0)
	roundSettings := s.node.GetCurrentRoundSettings()
	s.onNewRoundEvent(MakeNewRoundEvent(roundSettings.Round, roundSettings.FetchTransactions))

	externalEvents := s.node.Events()
	var nextPeerStateCh <-chan time.Time
	for {
		nextPeerStateTime := s.scheduler.nextDuration()
		if nextPeerStateTime != time.Duration(0) {
			nextPeerStateCh = s.clock.TimeoutAt(nextPeerStateTime)
		} else {
			nextPeerStateCh = nil
		}
		select {
		case ent := <-externalEvents:
			switch ent.eventType {
			case transactionPoolChangedEvent:
				s.onTransactionPoolChangedEvent(ent)
			case newRoundEvent:
				s.onNewRoundEvent(ent)
			}
			continue
		case <-nextPeerStateCh:
			s.evaluatePeerStateChanges(nextPeerStateTime)
			continue
		case incomingMsg := <-s.incomingMessagesCh:
			s.evaluateIncomingMessage(incomingMsg)
			continue
		case msgSent := <-s.outgoingMessagesCallbackCh:
			s.evaluateOutgoingMessage(msgSent)
			continue
		case <-s.nextOffsetRollingCh:
			s.rollOffsets()
			continue
		case <-serviceCtx.Done():
			return
		default:
		}

		s.node.NotifyState(StateMachineBlocked)
		select {
		case ent := <-externalEvents:
			s.node.NotifyState(StateMachineRunning)
			switch ent.eventType {
			case transactionPoolChangedEvent:
				s.onTransactionPoolChangedEvent(ent)
			case newRoundEvent:
				s.onNewRoundEvent(ent)
			}
		case <-nextPeerStateCh:
			s.node.NotifyState(StateMachineRunning)
			s.evaluatePeerStateChanges(nextPeerStateTime)
		case incomingMsg := <-s.incomingMessagesCh:
			s.node.NotifyState(StateMachineRunning)
			s.evaluateIncomingMessage(incomingMsg)
		case msgSent := <-s.outgoingMessagesCallbackCh:
			s.node.NotifyState(StateMachineRunning)
			s.evaluateOutgoingMessage(msgSent)
		case <-s.nextOffsetRollingCh:
			s.node.NotifyState(StateMachineRunning)
			s.rollOffsets()
		case <-serviceCtx.Done():
			s.node.NotifyState(StateMachineRunning)
			return
		}
	}
}

func (s *syncState) onTransactionPoolChangedEvent(ent Event) {
	newBeta := beta(ent.transactionPoolSize)
	// see if the newBeta is at least 20% smaller than the current one.
	if (s.lastBeta * 8 / 10) <= newBeta {
		// no, it's not.
		return
	}
	// yes, the number of transactions in the pool have changed dramatically since the last time.
	s.lastBeta = newBeta

	peers := make([]*Peer, 0, len(s.interruptablePeers))
	for peer := range s.interruptablePeers {
		peers = append(peers, peer)
		peer.state = peerStateHoldsoff

	}
	// reset the interruptablePeers table, since all it's members were made into holdsoff
	s.interruptablePeers = make(map[*Peer]bool)

	deadlineMonitor := s.clock.DeadlineMonitorAt(s.clock.Since() + sendMessagesTime)
	s.sendMessageLoop(deadlineMonitor, peers)

	currentTimeout := s.clock.Since()
	for _, peer := range peers {
		peerNext := s.scheduler.peerDuration(peer)
		if peerNext < currentTimeout {
			// shouldn't be, but let's reschedule it if this is the case.
			s.scheduler.schedulerPeer(peer, currentTimeout+s.lastBeta)
			continue
		}
		// given that peerNext is after currentTimeout, find out what's the difference, and divide by the beta.
		betaCount := (peerNext - currentTimeout) / s.lastBeta
		peerNext = currentTimeout + s.lastBeta*betaCount
		s.scheduler.schedulerPeer(peer, peerNext)
	}
}

// calculate the beta parameter, based on the transcation pool size.
func beta(txPoolSize int) time.Duration {
	if txPoolSize < 200 {
		txPoolSize = 200
	} else if txPoolSize > 10000 {
		txPoolSize = 10000
	}
	beta := 1.0 / (2 * 3.6923 * math.Exp(float64(txPoolSize)*0.00026))
	return time.Duration(float64(time.Second) * beta)

}

func (s *syncState) onNewRoundEvent(ent Event) {
	s.clock = s.clock.Zero().(timers.WallClock)
	peers := s.getPeers()
	newRoundPeers := peers
	if s.isRelay {
		// on relays, outgoing peers have a difference scheduling, which is based on the incoming message timing
		// rather then a priodic message transmission.
		newRoundPeers = incomingPeersOnly(newRoundPeers)
	}
	s.scheduler.scheduleNewRound(newRoundPeers)
	s.updatePeersRequestParams(peers)
	s.round = ent.roundSettings.Round
	s.fetchTransactions = ent.roundSettings.FetchTransactions
	s.nextOffsetRollingCh = s.clock.TimeoutAt(kickoffTime + 2*s.lastBeta)
}

func (s *syncState) evaluatePeerStateChanges(currentTimeout time.Duration) {
	peers := s.scheduler.nextPeers()
	if len(peers) == 0 {
		return
	}

	sendMessagePeers := 0
	if s.isRelay {
		// on relays, peers don't go into peerStateInterrupt; instead, they consistently remain in peerStateHoldsoff with double the beta.
		for _, peer := range peers {
			if peer.isOutgoing {
				// outgoing peers are "special", as they respond to messages rather then generating their own.
				// we need to figure the special state needed for "late bloom filter message"
				switch peer.state {
				case peerStateHoldsoff:
					// send a message, and reschedule for late bloom message.
					// prepare the send message array.
					peers[sendMessagePeers] = peer
					sendMessagePeers++
					peer.state = peerStateLateBloom
					// todo - find the proper way to reschedule that from the peer's data.
					s.scheduler.schedulerPeer(peer, peer.lastReceivedMessageTimestamp+peer.lastReceivedMessageNextMsgMinDelay/2)
				case peerStateLateBloom:
					// send just the bloom filter message.
					// how ?

					// reset peer state
					peer.state = peerStateStartup
				default:
					// this isn't expected, so we can just ignore this.
					// todo : log
				}
			} else {
				switch peer.state {
				case peerStateStartup:
					peer.state = peerStateHoldsoff
					fallthrough
				case peerStateHoldsoff:
					s.scheduler.schedulerPeer(peer, currentTimeout+s.lastBeta*2)
					// prepare the send message array.
					peers[sendMessagePeers] = peer
					sendMessagePeers++
				default: // peerStateInterrupt & peerStateLateBloom
					// this isn't expected, so we can just ignore this.
					// todo : log
				}
			}
		}
	} else {
		for _, peer := range peers {
			switch peer.state {
			case peerStateHoldsoff:
				peer.state = peerStateInterrupt
				s.scheduler.schedulerPeer(peer, currentTimeout+s.lastBeta)
				s.interruptablePeers[peer] = true
			case peerStateStartup:
				fallthrough
			case peerStateInterrupt:
				peer.state = peerStateHoldsoff
				s.scheduler.schedulerPeer(peer, currentTimeout+s.lastBeta)
				// prepare the send message array.
				peers[sendMessagePeers] = peer
				sendMessagePeers++
				delete(s.interruptablePeers, peer)
			default: // peerStateLateBloom
				// this isn't expected, so we can just ignore this.
				// todo : log
			}
		}
	}

	peers = peers[:sendMessagePeers]
	deadlineMonitor := s.clock.DeadlineMonitorAt(currentTimeout + sendMessagesTime)
	s.sendMessageLoop(deadlineMonitor, peers)
}

func (s *syncState) rollOffsets() {
	s.nextOffsetRollingCh = s.clock.TimeoutAt(s.clock.Since() + 2*s.lastBeta)
	s.requestsOffset++
}

func (s *syncState) getPeers() (result []*Peer) {
	peersInfo := s.node.GetPeers()
	updatedNetworkPeers := []interface{}{}
	updatedNetworkPeersSync := []*Peer{}
	// some of the network peers might not have a sync peer, so we need to create one for these.
	for _, peerInfo := range peersInfo {
		if peerInfo.TxnSyncPeer == nil {
			syncPeer := makePeer(peerInfo.NetworkPeer, peerInfo.IsOutgoing)
			peerInfo.TxnSyncPeer = syncPeer
			updatedNetworkPeers = append(updatedNetworkPeers, peerInfo.NetworkPeer)
			updatedNetworkPeersSync = append(updatedNetworkPeersSync, syncPeer)
		}
		result = append(result, peerInfo.TxnSyncPeer)
	}
	if len(updatedNetworkPeers) > 0 {
		s.node.UpdatePeers(updatedNetworkPeersSync, updatedNetworkPeers)
	}
	return result
}

func (s *syncState) updatePeersRequestParams(peers []*Peer) {
	for i, peer := range peers {
		if s.isRelay {
			// on relay, ask for all messages.
			peer.setLocalRequestParams(0, 1)
		} else {
			// on non-relay, ask for offset/modulator
			peer.setLocalRequestParams(uint64(i), uint64(len(peers)))
		}
	}
}
