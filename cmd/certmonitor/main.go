// Copyright (C) 2019 Algorand, Inc.
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
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/algorand/go-algorand/agreement"
	_ "github.com/algorand/go-algorand/cmd/certmonitor/pq"
	algodclient "github.com/algorand/go-algorand/daemon/algod/api/client"
	"github.com/algorand/go-algorand/data/basics"
	"github.com/algorand/go-algorand/data/bookkeeping"
	"github.com/algorand/go-algorand/logging"
	"github.com/algorand/go-algorand/network"
	"github.com/algorand/go-algorand/protocol"
	"github.com/algorand/go-algorand/rpcs"
	tools_network "github.com/algorand/go-algorand/tools/network"
	"github.com/algorand/websocket"
)

/*
const statusEndpoint = "http://r1.algorand.network:5160"
const statusToken = "8bef3da297740104ee50f823b0a9ef3df52e8d707655f22eeb6cbd4c5bcd1193"
*/
const statusEndpoint = "http://localhost:5160"
const statusToken = "11447faa00ad3e9414430a582601f7c0dc6a1f7dbe9a2cd29584414f373ca3fc"

// HTTPPeer ...
type HTTPPeer struct {
	rootURL string
	client  http.Client
	genesis string
}

// GetAddress is ...
func (p *HTTPPeer) GetAddress() string {
	return p.rootURL
}

// PrepareURL is ...
func (p *HTTPPeer) PrepareURL(x string) string {
	return strings.Replace(x, "{genesisID}", p.genesis, -1)
}

// GetHTTPClient ...
func (p *HTTPPeer) GetHTTPClient() *http.Client {
	return &p.client
}

// GetHTTPPeer ....
func (p *HTTPPeer) GetHTTPPeer() network.HTTPPeer {
	return p
}

func processBlockBytes(fetchedBuf []byte, r basics.Round, debugStr string) (blk *bookkeeping.Block, cert *agreement.Certificate, err error) {
	var decodedEntry rpcs.EncodedBlockCert
	err = protocol.Decode(fetchedBuf, &decodedEntry)
	if err != nil {
		err = fmt.Errorf("networkFetcher.FetchBlock(%d): cannot decode block from peer %v: %v", r, debugStr, err)
		return
	}

	if decodedEntry.Block.Round() != r {
		err = fmt.Errorf("networkFetcher.FetchBlock(%d): got wrong block from peer %v: wanted %v, got %v", r, debugStr, r, decodedEntry.Block.Round())
		return
	}

	if decodedEntry.Certificate.Round != r {
		err = fmt.Errorf("networkFetcher.FetchBlock(%d): got wrong cert from peer %v: wanted %v, got %v", r, debugStr, r, decodedEntry.Certificate.Round)
		return
	}
	return &decodedEntry.Block, &decodedEntry.Certificate, nil
}

func fetchBlocks(round basics.Round) (certMap map[string][]string, period int, step int) {
	certificates := make(map[string][]string)
	timeoutContext, cancelContextFunc := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelContextFunc()
	srvPhonebook, err := tools_network.ReadFromSRV("algobootstrap", "mainnet.algorand.network", "8.8.8.8")
	if err != nil {
		fmt.Printf("unable to retrieve phonebook entries.\n")
		return certificates, 0, 0
	}
	fmt.Printf("%d phonebook entries retrieved\n", len(srvPhonebook))
	fetchers := make([]rpcs.FetcherClient, 0)
	for _, entry := range srvPhonebook {
		httpPeer := &HTTPPeer{rootURL: entry, genesis: "mainnet-v1.0"}
		httpFetcher := rpcs.MakeHTTPFetcher(logging.Base(), httpPeer)
		fetchers = append(fetchers, httpFetcher)
	}

	var syncMutex sync.Mutex
	var wg sync.WaitGroup

	for _, fetcher := range fetchers {
		wg.Add(1)
		go func(fetcher rpcs.FetcherClient) {
			defer wg.Done()
			data, err := fetcher.GetBlockBytes(timeoutContext, round)
			if err != nil {
				fmt.Printf("Unable to get block %d from %s : %v\n", round, fetcher.Address(), err)
				return
			}
			_, cert, err := processBlockBytes(data, round, "")
			if err != nil {
				return
			}
			auth := make([]string, 0)
			for _, addr := range cert.Authenticators() {
				auth = append(auth, addr.String())
			}
			syncMutex.Lock()
			defer syncMutex.Unlock()
			certificates[fetcher.Address()] = auth
			period = int(cert.Period)
			step = int(cert.Step)
		}(fetcher)
	}
	wg.Wait()
	return certificates, period, step
}
func saveBlocksToDB(auths map[string][]string, round uint64, period int, step int, votes []voteSentEventData) {
	connStr := "postgres://tsachi:gogators@localhost/relays?sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		fmt.Printf("Unable to connect to database : %v\n", err)
	}
	defer db.Close()
	fmt.Printf("Database connection established\n")
	timeoutContext, cancelContextFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelContextFunc()
	updatedRows := 0
	tx, err := db.BeginTx(timeoutContext, nil)
	if err != nil {
		fmt.Printf("Unable to create transaction : %v\n", err)
	}
	commit := true
	for relay, authlist := range auths {
		for _, auth := range authlist {
			result, err := tx.ExecContext(timeoutContext, "INSERT INTO authenticators(relay, round, auth) VALUES($1, $2, $3)", relay, round, auth)
			if err != nil {
				fmt.Printf("Database update failed : %v\n", err)
				commit = false
				continue
			}
			rows, err := result.RowsAffected()
			if err != nil {
				fmt.Printf("Database update failed : %v\n", err)
				commit = false
				continue
			}
			if rows != 1 {
				fmt.Printf("expected to affect 1 row, affected %d\n", rows)
				commit = false
				continue
			}
			updatedRows++
		}
	}
	fmt.Printf("%d rows updated on database\n", updatedRows)

	_, err = tx.ExecContext(timeoutContext,
		"insert into authenticators_distribution(round, auth, dist) "+
			"select $1 as round, dist_auth.auth as auth, "+
			"(select count(*) from authenticators where round=$1 and auth=dist_auth.auth) * 1.0 "+
			"/ (select count(distinct relay) from authenticators where round=$1) as dist "+
			"from (select distinct auth from authenticators where round=$1) as dist_auth",
		round)
	if err != nil {
		fmt.Printf("Database update failed : %v\n", err)
		commit = false
	}

	_, err = tx.ExecContext(timeoutContext,
		"insert into rounds(round, relay_count, auth_count, period, step) "+
			"select $1 as round, "+
			"(select count(distinct relay) from authenticators where round=$1) as relay_count, "+
			"(select count(distinct auth) from authenticators where round=$1) as auth_count, $2, $3",
		round,
		period,
		step)
	if err != nil {
		fmt.Printf("Database update failed : %v\n", err)
		commit = false
	}

	_, err = tx.ExecContext(timeoutContext,
		"insert into roundrelays(round, relay, auth_count) "+
			"select $1, relay, count(auth) "+
			"from authenticators "+
			"where round=$1 "+
			"group by relay",
		round)
	if err != nil {
		fmt.Printf("Database update failed : %v\n", err)
		commit = false
	}

	for _, vote := range votes {
		result, err := tx.ExecContext(timeoutContext, "INSERT INTO votes(sendertelemetryid, timestamp, sender, round, period, step, weight) VALUES($1, $2, $3, $4, $5, $6, $7)", vote.Sender, vote.Timestamp, vote.Authenticator, vote.Round, vote.Period, vote.Step, vote.Weight)
		if err != nil {
			fmt.Printf("Database update failed : %v\n", err)
			commit = false
			continue
		}
		rows, err := result.RowsAffected()
		if err != nil {
			fmt.Printf("Database update failed : %v\n", err)
			commit = false
			continue
		}
		if rows != 1 {
			fmt.Printf("expected to affect 1 row, affected %d\n", rows)
			commit = false
			continue
		}
	}

	if commit {
		tx.Commit()
	} else {
		tx.Rollback()
	}
}
func certUpdateLoop() {
	url, err := url.Parse(statusEndpoint)
	if err != nil {
		fmt.Printf("unable to parse url : %v", err)
	}
	lastRound := uint64(0)
	for err == nil {

		fmt.Printf("Retrieving latest round...\n")
		restClient := algodclient.MakeRestClient(*url, statusToken)

		status, err2 := restClient.Status()
		if err2 != nil {
			continue
		}
		if status.LastRound == lastRound {
			continue
		}
		lastRound = status.LastRound
		updateRound := lastRound - 10
		fmt.Printf("Retrieving status for round %d\n", updateRound)
		relayauths, period, step := fetchBlocks(basics.Round(updateRound))
		fmt.Printf("%d certificates have been retrieved\n", len(relayauths))
		roundvotes := queryVotes(basics.Round(updateRound), time.Time{}, time.Time{})
		saveBlocksToDB(relayauths, updateRound, period, step, roundvotes)
	}

	fmt.Printf("Done!\n")
}

var websocketDialer = websocket.Dialer{
	Proxy:             http.ProxyFromEnvironment,
	HandshakeTimeout:  45 * time.Second,
	EnableCompression: false,
}

func getRelaySessionAssociation() (association map[string]string) {
	association = make(map[string]string)
	timeoutContext, cancelContextFunc := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelContextFunc()
	srvPhonebook, err := tools_network.ReadFromSRV("algobootstrap", "mainnet.algorand.network", "8.8.8.8")
	if err != nil {
		fmt.Printf("unable to retrieve phonebook entries.\n")
		return
	}
	fmt.Printf("%d phonebook entries retrieved\n", len(srvPhonebook))
	var syncMutex sync.Mutex
	var wg sync.WaitGroup

	for _, relay := range srvPhonebook {
		wg.Add(1)
		go func(ctx context.Context, relay string) {
			defer wg.Done()
			requestHeader := make(http.Header)
			requestHeader.Set("X-Algorand-Version", "1")
			requestHeader.Set("X-Algorand-NodeRandom", "1234")
			timeoutContext, cancelContextFunc := context.WithTimeout(ctx, time.Second*3)
			defer cancelContextFunc()
			conn, response, err := websocketDialer.DialContext(timeoutContext, "ws://"+relay+"/v1/mainnet-v1.0/gossip", requestHeader)
			if err != nil {
				fmt.Printf("failed to get gossip network info : %v\n", err)
				return
			}
			conn.Close()
			syncMutex.Lock()
			defer syncMutex.Unlock()
			if response != nil && response.Header != nil && len(response.Header["X-Algorand-Telid"]) > 0 {
				telemetry := response.Header["X-Algorand-Telid"][0]
				if len(telemetry) > 0 {
					association[relay] = telemetry
					fmt.Printf("%s => %s\n", relay, telemetry)
				} else {
					fmt.Printf("%s => Relay does not have telemetry enabled\n", relay)
				}
			}
		}(timeoutContext, relay)
	}
	wg.Wait()
	fmt.Printf("Relay telemetry association retrieval done.\n")
	return
}

func saveRelaySessionAssociation(association map[string]string) {
	if len(association) == 0 {
		return
	}

	connStr := "postgres://tsachi:gogators@localhost/relays?sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		fmt.Printf("Unable to connect to database : %v\n", err)
	}
	defer db.Close()
	fmt.Printf("Database connection established\n")
	timeoutContext, cancelContextFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelContextFunc()
	tx, err := db.BeginTx(timeoutContext, nil)
	if err != nil {
		fmt.Printf("Unable to create transaction : %v\n", err)
	}
	commit := true

	for relayName, telemetryID := range association {
		result, err := tx.ExecContext(timeoutContext, "INSERT INTO relaytelemetryid(telemetryid, relay) VALUES($1, $2) ON CONFLICT (telemetryid) DO UPDATE SET relay = $2", telemetryID, relayName)
		if err != nil {
			fmt.Printf("Database update failed : %v\n", err)
			commit = false
			break
		}
		rows, err := result.RowsAffected()
		if err != nil {
			fmt.Printf("Database update failed : %v\n", err)
			commit = false
			break
		}
		if rows != 1 {
			fmt.Printf("expected to affect 1 row, affected %d\n", rows)
			commit = false
			break
		}
	}

	if commit {
		tx.Commit()
	} else {
		tx.Rollback()
	}
}

func relaySessionGUIDLoop() {
	for {
		association := getRelaySessionAssociation()
		if len(association) > 0 {
			saveRelaySessionAssociation(association)
			time.Sleep(10 * time.Minute)
		} else {
			time.Sleep(10 * time.Second)
		}

	}
}
func getLastConnectionUpdate() (lastupdate time.Time, err error) {
	connStr := "postgres://tsachi:gogators@localhost/relays?sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		fmt.Printf("Unable to connect to database : %v\n", err)
	}
	defer db.Close()
	timeoutContext, cancelContextFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelContextFunc()
	err = db.QueryRowContext(timeoutContext, "select timevalue from dynamics where key='lastConnectionUpdate'").Scan(&lastupdate)
	switch {
	case err == sql.ErrNoRows:
		return time.Now().Add(-5 * 24 * time.Hour), nil
	case err != nil:
		fmt.Printf("getLastConnectionUpdate : %v\n", err)
		return time.Now(), err
	default:
		// we already have the desired result.
		return lastupdate, err
	}
}

func storeConnections(conn *esConnections, thisTimeStart, nextTimeStart time.Time) (err error) {
	connStr := "postgres://tsachi:gogators@localhost/relays?sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		fmt.Printf("Unable to connect to database : %v\n", err)
	}
	defer db.Close()
	timeoutContext, cancelContextFunc := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancelContextFunc()

	tx, err := db.BeginTx(timeoutContext, nil)
	if err != nil {
		fmt.Printf("Unable to create transaction : %v\n", err)
		return err
	}
	commit := true

	quantinizedTime := time.Date(
		thisTimeStart.Year(),
		thisTimeStart.Month(),
		thisTimeStart.Day(),
		thisTimeStart.Hour(),
		0, 0, 0, thisTimeStart.Location())

	_, err = tx.ExecContext(timeoutContext, "DELETE FROM connections where quanttime = $1", quantinizedTime)
	if err != nil {
		fmt.Printf("Database update failed : %v\n", err)
		commit = false
	}

	for _, node := range conn.nodes {
		for other := range node.conn {
			_, err = tx.ExecContext(timeoutContext, "INSERT INTO connections(quanttime, guid, address, name, otherguid, otheraddress, othername) VALUES($1, $2, $3, $4, $5, $6, $7)",
				quantinizedTime,
				node.guid,
				node.address,
				node.name,
				other.guid,
				other.address,
				other.name)
			if err != nil {
				fmt.Printf("Database update failed : %v\n", err)
				commit = false
			}
		}
	}

	//var result sql.Result
	_, err = tx.ExecContext(timeoutContext, "INSERT INTO dynamics(key, timevalue) VALUES('lastConnectionUpdate', $1) ON CONFLICT (key) DO UPDATE SET timevalue = $1", nextTimeStart)
	if err != nil {
		fmt.Printf("Database update failed : %v\n", err)
		commit = false
	}

	if commit {
		tx.Commit()
	} else {
		tx.Rollback()
		return fmt.Errorf("")
	}

	fmt.Printf("Stored connections for time window starting at %v\n", thisTimeStart)

	return nil
}

func updateConnections() {
	fmt.Printf("updating connections..\n")
	var connections esConnections
	for {
		lastupdate, err := getLastConnectionUpdate()
		if err != nil {
			fmt.Printf("getLastConnectionUpdate : %v..\n", err)
			time.Sleep(10 * time.Second)
			continue
		}

		if lastupdate.Add(10 * time.Minute).After(time.Now()) {
			fmt.Printf("too early; last update was at %v, sleeping..\n", lastupdate)
			time.Sleep(10 * time.Second)
			continue
		}
		err = queryConnections(lastupdate, lastupdate.Add(10*time.Minute), &connections)
		if err != nil {
			fmt.Printf("queryConnections : %v..\n", err)
			time.Sleep(10 * time.Second)
			continue
		}
		err = storeConnections(&connections, lastupdate, lastupdate.Add(10*time.Minute))
		if err != nil {
			fmt.Printf("storeConnections : %v..\n", err)
			time.Sleep(10 * time.Second)
			continue
		}
	}
}

func main() {

	go certUpdateLoop()
	go relaySessionGUIDLoop()
	go updateConnections()

	for {
		time.Sleep(time.Second)
	}
}
