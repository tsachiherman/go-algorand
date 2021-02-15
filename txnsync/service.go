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
	"context"
	"sync"

	"github.com/algorand/go-algorand/logging"
)

// Service is the transaction sync main service object.
type Service struct {
	nodeConn NodeConnector
	log      logging.Logger

	ctx       context.Context
	cancelCtx context.CancelFunc
	waitGroup sync.WaitGroup

	state syncState
}

// MakeTranscationSyncService creates a new Service object
func MakeTranscationSyncService(log logging.Logger, conn NodeConnector) *Service {
	return &Service{
		log:      log,
		nodeConn: conn,
	}
}

// Start starts the transaction sync
func (s *Service) Start() {
	s.ctx, s.cancelCtx = context.WithCancel(context.Background())
	s.waitGroup.Add(1)

	state := syncState{
		service: s,
		node:    s.nodeConn,
		log:     s.log,
	}
	go state.mainloop(s.ctx, &s.waitGroup)
}

// Stop stops the transaction sync
func (s *Service) Stop() {
	// cancel the context
	s.cancelCtx()
	// wait until the mainloop exists.
	s.waitGroup.Wait()
	// clear the context, as we won't be using it anymore.
	s.cancelCtx, s.ctx = nil, nil
}
