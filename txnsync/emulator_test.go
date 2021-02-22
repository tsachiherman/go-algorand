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
	"testing"
	"time"

	"github.com/algorand/go-algorand/data/basics"
)

type connectionSettings struct {
	uploadSpeed   uint64 // measured in bytes/second
	downloadSpeed uint64 // measured in bytes/second
	target        int    // node index in the networkConfiguration
}

type nodeConfiguration struct {
	outgoingConnections []connectionSettings
	name                string
	isRelay             bool
}

// networkConfiguration defines the nodes setup and their connections.
type networkConfiguration struct {
	nodes []nodeConfiguration
}

// initialTransactionsAllocation defines how many transaction ( and what their sizes ) would be.
type initialTransactionsAllocation struct {
	node              int // node index in the networkConfiguration
	transactionsCount int
	transactionSize   int
	expirationRound   basics.Round
}

// scenario defines the emulator test scenario, which includes the network configuration,
// initial transaction distribution, test duration, dynamic transactions creation as well
// as expected test outcomes.
type scenario struct {
	netConfig       networkConfiguration
	testDuration    time.Duration
	initialAlloc    []initialTransactionsAllocation
	expectedResults emulatorResult
}

func TestEmulatedNonRelayToRelayTransactionsExchange(t *testing.T) {
	testScenario := scenario{
		netConfig: networkConfiguration{
			nodes: []nodeConfiguration{
				{
					name:    "relay",
					isRelay: true,
				},
				{
					name: "node",
					outgoingConnections: []connectionSettings{
						{
							uploadSpeed:   1000000,
							downloadSpeed: 1000000,
							target:        0,
						},
					},
				},
			},
		},
		testDuration: 8500 * time.Millisecond,
		initialAlloc: []initialTransactionsAllocation{
			initialTransactionsAllocation{
				node:              1,
				transactionsCount: 1,
				transactionSize:   250,
				expirationRound:   basics.Round(5),
			},
		},
		expectedResults: emulatorResult{
			nodes: []nodeTransactions{
				{
					nodeTransaction{
						expirationRound: 5,
						transactionSize: 250,
					},
				},
				{
					nodeTransaction{
						expirationRound: 5,
						transactionSize: 250,
					},
				},
			},
		},
	}
	emulateScenario(t, testScenario)
}

func TestEmulatedRelayToNonRelayTransactionsExchange(t *testing.T) {
	testScenario := scenario{
		netConfig: networkConfiguration{
			nodes: []nodeConfiguration{
				{
					name:    "relay",
					isRelay: true,
				},
				{
					name: "node",
					outgoingConnections: []connectionSettings{
						{
							uploadSpeed:   1000000,
							downloadSpeed: 1000000,
							target:        0,
						},
					},
				},
			},
		},
		testDuration: 8500 * time.Millisecond,
		initialAlloc: []initialTransactionsAllocation{
			initialTransactionsAllocation{
				node:              0,
				transactionsCount: 1,
				transactionSize:   250,
				expirationRound:   basics.Round(5),
			},
		},
		expectedResults: emulatorResult{
			nodes: []nodeTransactions{
				{
					nodeTransaction{
						expirationRound: 5,
						transactionSize: 250,
					},
				},
				{
					nodeTransaction{
						expirationRound: 5,
						transactionSize: 250,
					},
				},
			},
		},
	}
	emulateScenario(t, testScenario)
}

func TestEmulatedOutgoingRelayToRelayTransactionsExchange(t *testing.T) {
	testScenario := scenario{
		netConfig: networkConfiguration{
			nodes: []nodeConfiguration{
				{
					name:    "incoming-relay",
					isRelay: true,
				},
				{
					name:    "outgoing-relay",
					isRelay: true,
					outgoingConnections: []connectionSettings{
						{
							uploadSpeed:   1000000,
							downloadSpeed: 1000000,
							target:        0,
						},
					},
				},
			},
		},
		testDuration: 8500 * time.Millisecond,
		initialAlloc: []initialTransactionsAllocation{
			initialTransactionsAllocation{
				node:              1,
				transactionsCount: 1,
				transactionSize:   250,
				expirationRound:   basics.Round(5),
			},
		},
		expectedResults: emulatorResult{
			nodes: []nodeTransactions{
				{
					nodeTransaction{
						expirationRound: 5,
						transactionSize: 250,
					},
				},
				{
					nodeTransaction{
						expirationRound: 5,
						transactionSize: 250,
					},
				},
			},
		},
	}
	emulateScenario(t, testScenario)
}

func TestEmulatedIncomingRelayToRelayTransactionsExchange(t *testing.T) {
	testScenario := scenario{
		netConfig: networkConfiguration{
			nodes: []nodeConfiguration{
				{
					name:    "incoming-relay",
					isRelay: true,
				},
				{
					name:    "outgoing-relay",
					isRelay: true,
					outgoingConnections: []connectionSettings{
						{
							uploadSpeed:   1000000,
							downloadSpeed: 1000000,
							target:        0,
						},
					},
				},
			},
		},
		testDuration: 8500 * time.Millisecond,
		initialAlloc: []initialTransactionsAllocation{
			initialTransactionsAllocation{
				node:              0,
				transactionsCount: 1,
				transactionSize:   250,
				expirationRound:   basics.Round(5),
			},
		},
		expectedResults: emulatorResult{
			nodes: []nodeTransactions{
				{
					nodeTransaction{
						expirationRound: 5,
						transactionSize: 250,
					},
				},
				{
					nodeTransaction{
						expirationRound: 5,
						transactionSize: 250,
					},
				},
			},
		},
	}
	emulateScenario(t, testScenario)
}
