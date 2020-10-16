// Copyright (C) 2019-2020 Algorand, Inc.
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

package agreement

import (
	"fmt"

	"github.com/algorand/go-deadlock"

	"github.com/algorand/go-algorand/config"
	"github.com/algorand/go-algorand/crypto"
	"github.com/algorand/go-algorand/data/basics"
	"github.com/algorand/go-algorand/data/committee"
	"github.com/algorand/go-algorand/protocol"
)

var _ = fmt.Printf

// accountTracker is an optimization layer between the agreement and the external
// ledger. The accountTracker is designed to optimize the ledger calls by caching
// accounts data by their validatity range, allowing optimistically no-lock aquisition
// of account data during signature validation.
type accountTracker struct {
	ledger LedgerReader
	log    serviceLogger

	roundTrackers []*roundAccountTracker
	baseRound     basics.Round
}

func makeAccountTracker(ledger LedgerReader, log serviceLogger) *accountTracker {
	return &accountTracker{
		ledger:        ledger,
		roundTrackers: []*roundAccountTracker{},
		baseRound:     0,
		log:           log,
	}
}

func (at *accountTracker) getRoundAccountTracker(rnd basics.Round) LedgerReader {
	// see if we have an account tracker for the requested round
	if rnd >= at.baseRound && rnd < at.baseRound+basics.Round(len(at.roundTrackers)) {
		// we do, return it.
		return at.roundTrackers[rnd-at.baseRound]
	}
	// we shouldn't get a request for rnd=0, so we will default to the upstream ledger.
	// if we get a request of a round before baseRound, we also default to the upstream ledger.
	if rnd == 0 || rnd < at.baseRound {
		return at.ledger
	}

	// ensure linearity - the roundTrackers need to contain all the rounds starting the base round.
	for newRound := at.baseRound + basics.Round(len(at.roundTrackers)); newRound <= rnd; newRound++ {
		if err := at.newRound(newRound); err != nil {
			// reset the entire cache pipeline. this is not expected to happen.
			at.roundTrackers = at.roundTrackers[:0]
			at.log.Warnf("accountTracker unable to create account tracker for round %d : %v", rnd, err)
			return at.ledger
		}
	}

	// now that the new round was created, we can return it.
	return at.roundTrackers[rnd-at.baseRound]
}

func (at *accountTracker) newRound(rnd basics.Round) error {
	// get the previous round tracker ( if we have it )
	var prevRoundTracker *roundAccountTracker
	previousRound := rnd.SubSaturate(1)
	if previousRound >= at.baseRound && previousRound < at.baseRound+basics.Round(len(at.roundTrackers)) {
		prevRoundTracker = at.roundTrackers[previousRound-at.baseRound]
	}

	roundAcctTracker, err := makeRoundAccountTracker(rnd, at, prevRoundTracker)
	if err != nil {
		return err
	}
	at.roundTrackers = append(at.roundTrackers, roundAcctTracker)

	// adjust history size
	cparams := roundAcctTracker.consensusParams

	backlogLength := int(2 * cparams.SeedRefreshInterval * cparams.SeedLookback)

	if len(at.roundTrackers) > backlogLength {
		at.roundTrackers = at.roundTrackers[1:]
		at.baseRound++
	}
	if len(at.roundTrackers) == 1 {
		at.baseRound = rnd
	}
	return nil
}

// ConsensusVersion returns the consensus version that is correct for the given round.
func (at *accountTracker) ConsensusVersion(rnd basics.Round) (protocol.ConsensusVersion, error) {
	return at.ledger.ConsensusVersion(rnd)
}

//msgp:ignore accountData
type accountData struct {
	// the AccountData contained here is the account data without rewards
	basics.AccountData
	validThrough basics.Round
}

// roundAccountTracker is the accounts tracker for a specific round. It implements the LedgerReader interface,
// and defaults all the uncached/unknown requests upstream
type roundAccountTracker struct {
	// parent account tracker
	accountTracker *accountTracker

	// the round number
	round basics.Round

	// the balance round ( i.e. round - SeedLookback )
	balanceRound basics.Round

	// the seed round ( i.e. round - 2 * SeedRefreshInterval * SeedLookback )
	seedRound basics.Round

	// the params round ( i.e. round - 2 )
	paramsRound basics.Round

	// cached round properties:

	// the seed at seedRound
	seed    committee.Seed
	hasSeed bool

	// the circulation at balanceRound
	circulation    basics.MicroAlgos
	hasCirculation bool

	// the consensusParams at paramsRound
	consensusParams config.ConsensusParams

	// the consensusParams at balanceRound
	balanceConsensusParams config.ConsensusParams

	// the rewards level at round balanceRound
	balanceRewardsLevel uint64

	// accounts contains the account data for accounts at balanceRound; it's being initialized in makeRoundAccountTracker and never being updated,
	// making it safe for concurrent use without any syncronization primitives.
	accounts map[basics.Address]*accountData

	// newAccounts contains the account data for accounts at balanceRound; it contains account data that wasn't available in accounts, and was added
	// during the signature authentication. This map requires taking the newAccountLock before using it.
	newAccounts map[basics.Address]*accountData

	// newAccountsLock is used to syncronize write access to newAccounts from multiple concurrent go-routines.
	newAccountsLock deadlock.RWMutex
}

func makeRoundAccountTracker(rnd basics.Round, at *accountTracker, prevRoundTracker *roundAccountTracker) (rndAcct *roundAccountTracker, err error) {
	rndAcct = &roundAccountTracker{
		round:          rnd,
		accountTracker: at,
		paramsRound:    ParamsRound(rnd),
		accounts:       make(map[basics.Address]*accountData),
		newAccounts:    make(map[basics.Address]*accountData),
	}
	rndAcct.consensusParams, err = at.ledger.ConsensusParams(rndAcct.paramsRound)
	if err != nil {
		return nil, err
	}
	rndAcct.balanceRound = balanceRound(rnd, rndAcct.consensusParams)
	rndAcct.balanceConsensusParams, err = at.ledger.ConsensusParams(rndAcct.balanceRound)
	if err != nil {
		return nil, err
	}
	rndAcct.seedRound = seedRound(rnd, rndAcct.consensusParams)
	if circulation, err := at.ledger.Circulation(rndAcct.balanceRound); err == nil {
		rndAcct.hasCirculation = true
		rndAcct.circulation = circulation
	}
	if seed, err := at.ledger.Seed(rndAcct.seedRound); err == nil {
		rndAcct.hasSeed = true
		rndAcct.seed = seed
	}
	rndAcct.balanceRewardsLevel, err = at.ledger.RewardsLevel(rndAcct.balanceRound)
	if err != nil {
		return nil, err
	}
	if prevRoundTracker != nil {
		// initialize accounts with the relevant account data from previous round.
		for addr, acctData := range prevRoundTracker.accounts {
			if acctData.validThrough < rnd {
				continue
			}
			rndAcct.accounts[addr] = acctData
		}
		prevRoundTracker.newAccountsLock.RLock()
		for addr, acctData := range prevRoundTracker.newAccounts {
			if acctData.validThrough < rnd {
				continue
			}
			rndAcct.accounts[addr] = acctData
		}
		prevRoundTracker.newAccountsLock.RUnlock()
	}
	return rndAcct, nil
}

// NextRound returns the first round for which no Block has been
// confirmed.
func (rt *roundAccountTracker) NextRound() basics.Round {
	return rt.accountTracker.ledger.NextRound()
}

// Wait returns a channel which fires when the specified round
// completes and is durably stored on disk.
func (rt *roundAccountTracker) Wait(rnd basics.Round) chan struct{} {
	return rt.accountTracker.ledger.Wait(rnd)
}

// Seed returns the VRF seed that was agreed upon in a given round.
func (rt *roundAccountTracker) Seed(rnd basics.Round) (committee.Seed, error) {
	if rnd == rt.seedRound && rt.hasSeed {
		return rt.seed, nil
	}
	return rt.accountTracker.ledger.Seed(rnd)
}

// Lookup returns the AccountData associated with some Address
// at the conclusion of a given round.
func (rt *roundAccountTracker) Lookup(rnd basics.Round, addr basics.Address) (basics.AccountData, error) {
	if rnd == rt.balanceRound {
		if acctData, has := rt.accounts[addr]; has {
			return acctData.WithUpdatedRewards(rt.balanceConsensusParams, rt.balanceRewardsLevel), nil
		}
		rt.newAccountsLock.RLock()
		if acctData, has := rt.newAccounts[addr]; has {
			rt.newAccountsLock.RUnlock()
			return acctData.WithUpdatedRewards(rt.balanceConsensusParams, rt.balanceRewardsLevel), nil
		}
		rt.newAccountsLock.RUnlock()
		// try to get the account data from the ledger.
		acctData, validThrough, err := rt.accountTracker.ledger.LookupWithoutRewards(rnd, addr)
		if err != nil {
			return acctData, err
		}
		rt.newAccountsLock.Lock()
		rt.newAccounts[addr] = &accountData{AccountData: acctData, validThrough: validThrough}
		rt.newAccountsLock.Unlock()
		return acctData.WithUpdatedRewards(rt.balanceConsensusParams, rt.balanceRewardsLevel), nil
	}
	return rt.accountTracker.ledger.Lookup(rnd, addr)
}

// Circulation returns the total amount of money in circulation at the
// conclusion of a given round.
func (rt *roundAccountTracker) Circulation(rnd basics.Round) (basics.MicroAlgos, error) {
	if rnd == rt.balanceRound && rt.hasCirculation {
		return rt.circulation, nil
	}
	return rt.accountTracker.ledger.Circulation(rnd)
}

// LookupDigest returns the Digest of the entry that was agreed on in a
// given round.
func (rt *roundAccountTracker) LookupDigest(rnd basics.Round) (crypto.Digest, error) {
	return rt.accountTracker.ledger.LookupDigest(rnd)
}

// ConsensusParams returns the consensus parameters that are correct
// for the given round.
func (rt *roundAccountTracker) ConsensusParams(rnd basics.Round) (config.ConsensusParams, error) {
	if rnd == rt.paramsRound {
		return rt.consensusParams, nil
	}
	return rt.accountTracker.ledger.ConsensusParams(rnd)
}

// ConsensusVersion returns the consensus version that is correct
// for the given round.
func (rt *roundAccountTracker) ConsensusVersion(rnd basics.Round) (protocol.ConsensusVersion, error) {
	return rt.accountTracker.ledger.ConsensusVersion(rnd)
}

// RewardsLevel returns the rewards level agreed upon when the given
// Round was added to the ledger.
func (rt *roundAccountTracker) RewardsLevel(rnd basics.Round) (uint64, error) {
	return rt.accountTracker.ledger.RewardsLevel(rnd)
}

// LookupWithoutRewards returns the AccountData associated with some
// Address at the conclusion of a given round without applying
// the rewards to that account data. The function also returns
// the last round at which the account data would remain the
// same beyond the requested round number.
func (rt *roundAccountTracker) LookupWithoutRewards(rnd basics.Round, addr basics.Address) (basics.AccountData, basics.Round, error) {
	return rt.accountTracker.ledger.LookupWithoutRewards(rnd, addr)
}
