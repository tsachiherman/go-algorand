// Copyright (C) 2019 Algorand, Inc.
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

package telemetryspec

import (
	"time"
)

// Telemetry Events

// Event is the type used to identify telemetry events
// We want these to be stable and easy to find / document so we can create queries against them.
type Event string

// StartupEvent event
const StartupEvent Event = "Startup"

// StartupEventDetails contains details for the StartupEvent
type StartupEventDetails struct {
	Version    string
	CommitHash string
	Branch     string
	Channel    string
	Instance   string
}

// HeartbeatEvent is sent periodically to indicate node is running
const HeartbeatEvent Event = "Heartbeat"

// HeartbeatEventDetails contains details for the StartupEvent
type HeartbeatEventDetails struct {
	Metrics map[string]string
}

// CatchupStartEvent event
const CatchupStartEvent Event = "CatchupStart"

// CatchupStartEventDetails contains details for the CatchupStartEvent
type CatchupStartEventDetails struct {
	StartRound uint64
}

// CatchupStopEvent event
const CatchupStopEvent Event = "CatchupStop"

// CatchupStopEventDetails contains details for the CatchupStopEvent
type CatchupStopEventDetails struct {
	StartRound uint64
	EndRound   uint64
	Time       time.Duration
	InitSync   bool
}

// ShutdownEvent event
const ShutdownEvent Event = "Shutdown"

// BlockAcceptedEvent event
const BlockAcceptedEvent Event = "BlockAccepted"

// BlockAcceptedEventDetails contains details for the BlockAcceptedEvent
type BlockAcceptedEventDetails struct {
	Address string
	Hash    string
	Round   uint64
}

// TopAccountsEvent event
const TopAccountsEvent Event = "TopAccounts"

// TopAccountEventDetails contains details for the BlockAcceptedEvent
type TopAccountEventDetails struct {
	Round              uint64
	OnlineAccounts     []map[string]interface{}
	OnlineCirculation  uint64
	OfflineCirculation uint64
}

// AccountRegisteredEvent event
const AccountRegisteredEvent Event = "AccountRegistered"

// AccountRegisteredEventDetails contains details for the AccountRegisteredEvent
type AccountRegisteredEventDetails struct {
	Address string
}

// PartKeyRegisteredEvent event
const PartKeyRegisteredEvent Event = "PartKeyRegistered"

// PartKeyRegisteredEventDetails contains details for the PartKeyRegisteredEvent
type PartKeyRegisteredEventDetails struct {
	Address    string
	FirstValid uint64
	LastValid  uint64
}

// BlockProposedEvent event
const BlockProposedEvent Event = "BlockProposed"

// BlockProposedEventDetails contains details for the BlockProposedEvent
type BlockProposedEventDetails struct {
	Address string
	Hash    string
	Round   uint64
	Period  uint64
	Step    uint64
}

// NewPeriodEvent event
const NewPeriodEvent Event = "NewPeriod"

// NewRoundPeriodDetails contains details for every new round or new period
// We explicitly log local time even though a timestamp is generated by logger.
type NewRoundPeriodDetails struct {
	OldRound  uint64
	OldPeriod uint64
	OldStep   uint64
	NewRound  uint64
	NewPeriod uint64
	NewStep   uint64
	LocalTime time.Time
}

// VoteSentEvent event
const VoteSentEvent Event = "VoteSent"

// VoteAcceptedEvent event
const VoteAcceptedEvent Event = "VoteAccepted"

// VoteEventDetails contains details for the VoteSentEvent
type VoteEventDetails struct {
	Address   string
	Hash      string
	Round     uint64
	Period    uint64
	Step      uint64
	Weight    uint64
	Recovered bool
}

// VoteRejectedEvent event
const VoteRejectedEvent Event = "VoteRejected"

// VoteRejectedEventDetails contains details for the VoteSentEvent
type VoteRejectedEventDetails struct {
	VoteEventDetails
	Reason string
}

// EquivocatedVoteEvent event
const EquivocatedVoteEvent Event = "EquivocatedVoteEvent"

// EquivocatedVoteEventDetails contains details for the EquivocatedVoteEvent
type EquivocatedVoteEventDetails struct {
	VoterAddress          string
	ProposalHash          string
	Round                 uint64
	Period                uint64
	Step                  uint64
	Weight                uint64
	PreviousProposalHash1 string
	PreviousProposalHash2 string
}

// ConnectPeerEvent event
const ConnectPeerEvent Event = "ConnectPeer"

// PeerEventDetails contains details for the ConnectPeerEvent
type PeerEventDetails struct {
	Address      string
	HostName     string
	Incoming     bool
	InstanceName string
}

// ConnectPeerFailEvent event
const ConnectPeerFailEvent Event = "ConnectPeerFail"

// ConnectPeerFailEventDetails contains details for the ConnectPeerFailEvent
type ConnectPeerFailEventDetails struct {
	Address      string
	HostName     string
	Incoming     bool
	InstanceName string
	Reason       string
}

// DisconnectPeerEvent event
const DisconnectPeerEvent Event = "DisconnectPeer"

// DisconnectPeerEventDetails contains details for the DisconnectPeerEvent
type DisconnectPeerEventDetails struct {
	PeerEventDetails
	Reason string
}

// ErrorOutputEvent event
const ErrorOutputEvent Event = "ErrorOutput"

// ErrorOutputEventDetails contains details for ErrorOutputEvent
type ErrorOutputEventDetails struct {
	Output string
	Error  string
}

// DeadManTriggeredEvent event
const DeadManTriggeredEvent Event = "DeadManTriggered"

// DeadManTriggeredEventDetails contains details for DeadManTriggeredEvent
type DeadManTriggeredEventDetails struct {
	Timeout      int64
	CurrentBlock uint64
	GoRoutines   string
}

// BlockStatsEvent event
const BlockStatsEvent Event = "BlockStats"

// BlockStatsEventDetails contains details for BlockStatsEvent
type BlockStatsEventDetails struct {
	Hash                string
	OriginalProposer    string
	Round               uint64
	Transactions        uint64
	ActiveUsers         uint64
	AgreementDurationMs uint64
	NetworkDowntimeMs   uint64
}
