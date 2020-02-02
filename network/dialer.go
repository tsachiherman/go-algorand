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

package network

import (
	"context"
	"net"
)

// Dialer establish tcp-level connection with the destination
type Dialer struct {
	phonebook   *MultiPhonebook
	innerDialer net.Dialer
}

// Dial connects to the address on the named network.
func (d *Dialer) Dial(network, address string) (net.Conn, error) {

	_, _, provisionalTime := d.phonebook.WaitForConnectionTime(address)	
	conn, err := d.innerDialer.Dial(network, address)
	d.phonebook.UpdateConnectionTime(address, provisionalTime)

	return conn, err
}

// DialContext connects to the address on the named network using the provided context.
func (d *Dialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {

	_, _, provisionalTime := d.phonebook.WaitForConnectionTime(address)
	conn, err := d.innerDialer.DialContext(ctx, network, address)
	d.phonebook.UpdateConnectionTime(address, provisionalTime)

	return conn, err
}
