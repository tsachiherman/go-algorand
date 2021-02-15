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
	"github.com/algorand/go-algorand/data/basics"
	"github.com/algorand/go-algorand/data/transactions"
)

const txnBlockMessageVersion = 1

type transactionBlockMessage struct {
	_struct struct{} `codec:",omitempty,omitemptyarray"`

	version              int                     `codec:"v"`
	round                basics.Round            `codec:"r"`
	txnBloomFilter       encodedBloomFilter      `codec:"b"`
	updatedRequestParams requestParams           `codec:"p"`
	transactionGroups    packedTransactionGroups `codec:"g"`
	msgSync              timingParams            `codec:"t"`
}

type encodedBloomFilter struct {
	_struct struct{} `codec:",omitempty,omitemptyarray"`

	bloomFilterType byte          `codec:"t"`
	encodingParams  requestParams `codec:"p"`
	shuffler        byte          `codec:"s"`
	bloomFilter     []byte        `codec:"f"`
}

type requestParams struct {
	_struct struct{} `codec:",omitempty,omitemptyarray"`

	offset    byte `codec:"o"`
	modulator byte `codec:"m"`
}

type packedTransactionGroups struct {
	_struct struct{} `codec:",omitempty,omitemptyarray"`

	transactionsGroup [][]transactions.SignedTxn `codec:"g"`
}

type timingParams struct {
	_struct struct{} `codec:",omitempty,omitemptyarray"`

	refTxnBlockMsgSeq   uint32   `codec:"s"`
	responseElapsedTime uint64   `codec:"r"`
	acceptedMsgSeq      []uint32 `codec:"a"`
	nextMsgMinDelay     uint64   `codec:"m"`
}
