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

package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/algorand/go-algorand/crypto"
	"github.com/algorand/go-algorand/data/basics"
)

var generateKeyfile string
var generatePubkeyfile string

func init() {
	generateCmd.Flags().StringVarP(&generateKeyfile, "keyfile", "f", "", "Private key filename")
	generateCmd.Flags().StringVarP(&generatePubkeyfile, "pubkeyfile", "p", "", "Public key filename")
}

// https://stackoverflow.com/a/50285590/356849
func toUint11Array(arr []byte) []uint32 {
	var buffer uint32
	var numberOfBit uint32
	var output []uint32

	for i := 0; i < len(arr); i++ {
		// prepend bits to buffer
		buffer |= uint32(arr[i]) << numberOfBit
		numberOfBit += 8

		// if there enough bits, extract 11bit number
		if numberOfBit >= 11 {
			// 0x7FF is 2047, the max 11 bit number
			output = append(output, buffer&0x7ff)

			// drop chunk from buffer
			buffer = buffer >> 11
			numberOfBit -= 11
		}

	}

	if numberOfBit != 0 {
		output = append(output, buffer&0x7ff)
	}
	return output
}

func toByteArray(arr []uint32) []byte {
	var buffer uint32
	var numberOfBits uint32
	var output []byte

	for i := 0; i < len(arr); i++ {
		buffer |= uint32(arr[i]) << numberOfBits
		numberOfBits += 11

		for numberOfBits >= 8 {
			output = append(output, byte(buffer&0xff))
			buffer >>= 8
			numberOfBits -= 8
		}
	}

	if numberOfBits != 0 {
		output = append(output, byte(buffer))
	}

	return output
}

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate key",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, _ []string) {
		var seed crypto.Seed
		crypto.RandBytes(seed[:])

		i := 0
		var mnemonic string
		for {
			mnemonic = computeMnemonic(seed)
			if len(strings.Split(mnemonic, " ")[i]) != 4 {

				uintArray := toUint11Array(seed[:])
				uintArray[i] = uint32(crypto.RandUint64() % 2048)

				copy(seed[:], toByteArray(uintArray))
				continue
			}
			i++
			if i > 23 {
				break
			}
		}

		key := crypto.GenerateSignatureSecrets(seed)
		publicKeyChecksummed := basics.Address(key.SignatureVerifier).String()

		fmt.Printf("Private key mnemonic: %s\n", mnemonic)
		fmt.Printf("Public key: %s\n", publicKeyChecksummed)

		if generateKeyfile != "" {
			writePrivateKey(generateKeyfile, seed)
		}

		if generatePubkeyfile != "" {
			writePublicKey(generatePubkeyfile, publicKeyChecksummed)
		}
	},
}
