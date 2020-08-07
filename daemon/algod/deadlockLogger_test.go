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

package algod

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/algorand/go-deadlock"

	"github.com/algorand/go-algorand/logging"
)

type lockedLogger struct {
	logging.Logger
	lockedMu deadlock.Mutex
}

func (ll *lockedLogger) Error(a ...interface{}) {
	ll.lockedMu.Lock()
	ll.Logger.Error(a...)
	ll.lockedMu.Unlock()
}

func (ll *lockedLogger) Panic(a ...interface{}) {
	ll.lockedMu.Lock()
	ll.Logger.Panic(a...)
	ll.lockedMu.Unlock()
}

func TestDeadlockLoggerBinding(t *testing.T) {
	logger := &lockedLogger{
		Logger: logging.Base(),
	}
	outFile, err := ioutil.TempFile(os.TempDir(), "loggingoutput")
	require.NoError(t, err)
	defer outFile.Close()

	setupDeadlockLogger(logger, outFile)
	var mu deadlock.RWMutex
	mu.RLock()
	// take the mutex lock.
	logger.lockedMu.Lock()
	defer logger.lockedMu.Unlock()

	// the logger is now blocking the Error/Panic calls, so that the deadlock library is blocked from
	// reporting the error directly to the logger. this would ensure that the "binding" would separate
	// the logging internal locks from the deadlock library locks.
	mu.RLock()

}
