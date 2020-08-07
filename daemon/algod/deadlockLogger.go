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
	"bytes"
	"fmt"
	"os"
	"runtime"

	"github.com/algorand/go-deadlock"

	"github.com/algorand/go-algorand/logging"
)

type dumpLogger struct {
	logging.Logger
	*bytes.Buffer
}

func (logger *dumpLogger) dumpAsync(callstackBuffer []byte) {
	deadlockMessageBytes := logger.Bytes()
	// create a copy of the deadlockMessageBytes, so that if it gets modified it won't affect the printing.
	// strings in go are immutable, so it would create a copy.
	message := string(deadlockMessageBytes)
	go func(message string, callstack string) {
		logger.Error(message)
		logger.Panic("potential deadlock detected here : %v", callstackBuffer)
	}(message, string(callstackBuffer))

}

func setupDeadlockLogger(syslogger logging.Logger, stdErr *os.File) {
	var logger = dumpLogger{Logger: syslogger, Buffer: bytes.NewBuffer(make([]byte, 0))}
	deadlock.Opts.LogBuf = logger
	deadlock.Opts.OnPotentialDeadlock = func() {
		// Capture all goroutine stacks and log to stderr
		var callstackBuffer []byte
		bufferSize := 256 * 1024
		for {
			callstackBuffer = make([]byte, bufferSize)
			if runtime.Stack(callstackBuffer, true) < bufferSize {
				break
			}
			bufferSize *= 2
		}
		fmt.Fprintln(stdErr, string(callstackBuffer))

		logger.dumpAsync(callstackBuffer)
	}
}
