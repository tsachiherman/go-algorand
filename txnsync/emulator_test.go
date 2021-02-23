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

func TestEmulatedTrivialTransactionsExchange(t *testing.T) {
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
	t.Run("NonRelay_To_Relay", func(t *testing.T) {
		testScenario.netConfig.nodes[0].name = "relay"
		testScenario.netConfig.nodes[0].isRelay = true
		testScenario.netConfig.nodes[1].name = "node"
		testScenario.initialAlloc[0].node = 1
		emulateScenario(t, testScenario)
	})
	t.Run("Relay_To_NonRelay", func(t *testing.T) {
		testScenario.netConfig.nodes[0].name = "relay"
		testScenario.netConfig.nodes[0].isRelay = true
		testScenario.netConfig.nodes[1].name = "node"
		testScenario.initialAlloc[0].node = 0
		emulateScenario(t, testScenario)
	})
	t.Run("OutgoingRelay_To_IncomingRelay", func(t *testing.T) {
		testScenario.netConfig.nodes[0].name = "incoming-relay"
		testScenario.netConfig.nodes[0].isRelay = true
		testScenario.netConfig.nodes[1].name = "outgoing-relay"
		testScenario.netConfig.nodes[1].isRelay = true
		testScenario.initialAlloc[0].node = 1
		emulateScenario(t, testScenario)
	})
	t.Run("OutgoingRelay_To_IncomingRelay", func(t *testing.T) {
		testScenario.netConfig.nodes[0].name = "incoming-relay"
		testScenario.netConfig.nodes[0].isRelay = true
		testScenario.netConfig.nodes[1].name = "outgoing-relay"
		testScenario.netConfig.nodes[1].isRelay = true
		testScenario.initialAlloc[0].node = 0
		emulateScenario(t, testScenario)
	})
}

func TestEmulatedTwoNodesToRelaysTransactionsExchange(t *testing.T) {
	// this test creates the following network mode:
	//
	//       relay1 ---------->  relay2
	//          ^                   ^
	//          |                   |
	//        node1               node2
	//

	testScenario := scenario{
		netConfig: networkConfiguration{
			nodes: []nodeConfiguration{
				{
					name:    "relay1",
					isRelay: true,
				},
				{
					name:    "relay2",
					isRelay: true,
					outgoingConnections: []connectionSettings{
						{
							uploadSpeed:   1000000,
							downloadSpeed: 1000000,
							target:        0,
						},
					},
				},
				{
					name: "node1",
					outgoingConnections: []connectionSettings{
						{
							uploadSpeed:   1000000,
							downloadSpeed: 1000000,
							target:        0,
						},
					},
				},
				{
					name: "node2",
					outgoingConnections: []connectionSettings{
						{
							uploadSpeed:   1000000,
							downloadSpeed: 1000000,
							target:        1,
						},
					},
				},
			},
		},
		testDuration: 8500 * time.Millisecond,
		initialAlloc: []initialTransactionsAllocation{
			initialTransactionsAllocation{
				node:              2,
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
