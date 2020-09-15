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

package ledger

import (
	"github.com/algorand/go-algorand/data/basics"
)

type accountsCacheEntry struct {
	prev, next *accountsCacheEntry
	data       basics.AccountData
	addr       basics.Address
}

// accountsCache is a caching structure used to maintain a cache of the most recently used cached accounts.
type accountsCache struct {
	size           int
	accountsLookup map[basics.Address]*accountsCacheEntry
	entries        []accountsCacheEntry
	first, last    *accountsCacheEntry
	nextEntry      int
}

func makeAccountsCache(size int) *accountsCache {
	return &accountsCache{
		size:           size,
		accountsLookup: make(map[basics.Address]*accountsCacheEntry, size),
		entries:        make([]accountsCacheEntry, size, size),
	}
}

func (ac *accountsCache) get(addr basics.Address) (data basics.AccountData, have bool) {
	if entry, has := ac.accountsLookup[addr]; has {
		return entry.data, true
	}
	return basics.AccountData{}, false
}

func (ac *accountsCache) add(addr basics.Address, data basics.AccountData) {
	var entry *accountsCacheEntry
	var has bool
	if entry, has = ac.accountsLookup[addr]; has {
		// promote the entry to the top of the list:

		// extract the entry out of the linked list
		if entry.prev != nil {
			entry.prev.next = entry.next
		}
		if entry.next != nil {
			entry.next.prev = entry.prev
		}
		if ac.first == entry {
			ac.first = entry.next
		}
		if ac.last == entry {
			ac.last = entry.prev
		}
	} else {
		// is the cache already full ? if so, discard the least recently used.
		if ac.nextEntry >= ac.size-1 {
			// we don't have any more entries available. discard the least recently used.
			entry = ac.first
			ac.first = entry.next
			entry.next = nil
			entry.prev = nil
			delete(ac.accountsLookup, entry.addr)
		} else {
			entry = &ac.entries[ac.nextEntry]
			ac.nextEntry++
		}
		ac.accountsLookup[addr] = entry
	}

	// add as the last entry on the linked list.
	entry.data = data
	entry.addr = addr
	if ac.last == nil {
		ac.last = entry
		ac.first = entry
	} else {
		entry.next = nil
		entry.prev = ac.last
		ac.last = entry
	}
	return
}

func (ac *accountsCache) update(addr basics.Address, data basics.AccountData) {
	var entry *accountsCacheEntry
	var has bool
	if entry, has = ac.accountsLookup[addr]; !has {
		return
	}
	entry.data = data

	// promote the entry to the top of the list:

	// extract the entry out of the linked list
	if entry.prev != nil {
		entry.prev.next = entry.next
	}
	if entry.next != nil {
		entry.next.prev = entry.prev
	}
	if ac.first == entry {
		ac.first = entry.next
	}
	if ac.last == entry {
		ac.last = entry.prev
	}

	// add as the last entry on the linked list.
	entry.data = data
	if ac.last == nil {
		ac.last = entry
		ac.first = entry
	} else {
		entry.next = nil
		entry.prev = ac.last
		ac.last = entry
	}
	return
}
