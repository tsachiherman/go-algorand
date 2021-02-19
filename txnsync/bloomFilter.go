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

	"github.com/algorand/go-algorand/util/bloom"
)

var errInvalidBloomFilterEncoding = errors.New("invalid bloom filter encoding")

//msgp:ignore bloomFilterTypes
type bloomFilterTypes byte

const (
	invalidBloomFilter bloomFilterTypes = iota
	multiHashBloomFilter
	// xorBloomFilter - todo.
)

type bloomFilter struct {
	encodingParams requestParams

	filter *bloom.Filter
}

func decodeBloomFilter(enc encodedBloomFilter) (outFilter bloomFilter, err error) {
	switch bloomFilterTypes(enc.bloomFilterType) {
	case multiHashBloomFilter:
	default:
		return bloomFilter{}, errInvalidBloomFilterEncoding
	}

	outFilter.filter, err = bloom.UnmarshalBinary(enc.bloomFilter)
	if err != nil {
		return bloomFilter{}, err
	}
	return outFilter, nil
}

func (bf *bloomFilter) encode() (out encodedBloomFilter) {
	out.bloomFilterType = byte(multiHashBloomFilter)
	out.encodingParams = bf.encodingParams
	out.bloomFilter, _ = bf.filter.MarshalBinary()
	return
}
