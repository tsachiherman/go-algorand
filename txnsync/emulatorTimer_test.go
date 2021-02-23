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
	"time"

	"github.com/algorand/go-deadlock"

	"github.com/algorand/go-algorand/util/timers"
)

// guidedClock implements the WallClock interface
type guidedClock struct {
	zero     time.Time
	adv      time.Duration
	timers   map[time.Duration]chan time.Time
	children []*guidedClock
	mu       deadlock.Mutex
}

func makeGuidedClock() *guidedClock {
	return &guidedClock{
		zero: time.Now(),
	}
}
func (g *guidedClock) Zero() timers.Clock {
	// the real monotonic clock doesn't return the same clock object, which is fine.. but for our testing
	// we want to keep the same clock object so that we can tweak with it.
	child := &guidedClock{
		zero: g.zero.Add(g.adv),
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.children = append(g.children, child)
	return child
}

func (g *guidedClock) TimeoutAt(delta time.Duration) <-chan time.Time {
	if delta <= g.adv {
		c := make(chan time.Time, 1)
		close(c)
		return c
	}
	if g.timers == nil {
		g.timers = make(map[time.Duration]chan time.Time)
	}
	c, has := g.timers[delta]
	if has {
		return c
	}
	c = make(chan time.Time, 1)
	g.timers[delta] = c
	return c
}

func (g *guidedClock) Encode() []byte {
	return []byte{}
}
func (g *guidedClock) Decode([]byte) (timers.Clock, error) {
	return &guidedClock{}, nil
}

func (g *guidedClock) Since() time.Duration {
	return g.adv
}

func (g *guidedClock) DeadlineMonitorAt(at time.Duration) timers.DeadlineMonitor {
	return timers.MakeMonotonicDeadlineMonitor(g, at)
}

func (g *guidedClock) Advance(adv time.Duration) {
	g.adv += adv

	expiredClocks := make(map[time.Duration]chan time.Time)
	// find all the expired clocks.
	for delta, ch := range g.timers {
		if delta < g.adv {
			expiredClocks[delta] = ch
		}
	}
	// remove from map
	for delta := range expiredClocks {
		delete(g.timers, delta)
	}
	// fire expired clocks
	for _, ch := range expiredClocks {
		ch <- g.zero.Add(g.adv)
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, child := range g.children {
		child.Advance(adv)
	}
}
