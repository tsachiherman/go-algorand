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
	"container/heap"
	"errors"
)

var errHeapEmpty = errors.New("message ordering heap is empty")
var errHeapReachedCapacity = errors.New("message ordering heap reached capacity")

const messageOrderingHeapLimit = 128

type messageHeapItem struct {
	blockMsg              transactionBlockMessage
	encodedBlockMsgLength int
	sequenceNumber        uint64
}

type messageOrderingHeap struct {
	messages        []messageHeapItem
	pendingMessages chan messageHeapItem
}

func makeMessageOrderingHeap() *messageOrderingHeap {
	return &messageOrderingHeap{
		pendingMessages: make(chan messageHeapItem, messageOrderingHeapLimit),
	}
}

// Push implements heap.Interface
func (p *messageOrderingHeap) Push(x interface{}) {
	entry := x.(messageHeapItem)
	p.messages = append(p.messages, entry)
}

// Pop implements heap.Interface
func (p *messageOrderingHeap) Pop() interface{} {
	end := len(p.messages) - 1
	res := p.messages[end]
	p.messages[end] = messageHeapItem{}
	p.messages = p.messages[0:end]
	return res
}

// Len implements heap.Interface
func (p *messageOrderingHeap) Len() int {
	return len(p.messages)
}

// Swap implements heap.Interface
func (p *messageOrderingHeap) Swap(i, j int) {
	p.messages[i], p.messages[j] = p.messages[j], p.messages[i]
}

// Less implements heap.Interface
func (p *messageOrderingHeap) Less(i, j int) bool {
	return p.messages[i].sequenceNumber < p.messages[j].sequenceNumber
}

func (p *messageOrderingHeap) enqueue(blockMsg transactionBlockMessage, sequenceNumber uint64, encodedBlockMsgLength int) error {
	select {
	case p.pendingMessages <- messageHeapItem{blockMsg: blockMsg, sequenceNumber: sequenceNumber, encodedBlockMsgLength: encodedBlockMsgLength}:
		return nil
	default:
		return errHeapReachedCapacity
	}
}

func (p *messageOrderingHeap) processPending() {
	for {
		select {
		case pendingMsg, ok := <-p.pendingMessages:
			if !ok {
				return
			}
			heap.Push(p, pendingMsg)
		default:
			return
		}
	}
}

func (p *messageOrderingHeap) peekSequence() (sequenceNumber uint64, err error) {
	p.processPending()
	if len(p.messages) == 0 {
		return 0, errHeapEmpty
	}
	return p.messages[0].sequenceNumber, nil
}

func (p *messageOrderingHeap) pop() (blockMsg transactionBlockMessage, encodedBlockMsgLength int, err error) {
	p.processPending()
	if len(p.messages) == 0 {
		return transactionBlockMessage{}, 0, errHeapEmpty
	}
	entry := heap.Pop(p).(messageHeapItem)
	return entry.blockMsg, entry.encodedBlockMsgLength, nil
}