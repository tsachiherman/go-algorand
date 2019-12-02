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
	"encoding/base32"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/algorand/go-algorand/crypto"
	"github.com/algorand/go-algorand/data/basics"
)

//curl -XPOST "https://elastic:JUO4Mn33qd6mFYC8aQA6MLQb@1ae9f9654b25441090fe5c48c833b95a.us-east-1.aws.found.io:9243/stable-mainnet-v1.0/_search" -H 'Content-Type: application/json' -d'
var voteQueryTemplate = `
{
  "size": 5000,
  "query": {
    "bool": {
      "must": [
        {
          "bool": {
            "should": [
              {
                "match": {
                  "Data.details.Round": {
                    "query": {Round}
                  }
                }
              },
              {
                "match": {
                  "Message.keyword": "/Agreement/VoteSent"
                }
              }
            ],
            "minimum_should_match": 2
          }
        },
        {
          "range": {
            "@timestamp": {
              "from": "{FromDate}",
              "to": "{ToDate}",
              "include_lower": true,
              "include_upper": true,
              "boost": 1
            }
          }
        }
      ]
    }
  }
}`

var connectionQueryTemplate = `
{
	"size": 5000,
	"query": {
	  "bool": {
		"must": [
		  {
			"bool": {
			  "should": [
				{
				  "match": {
					"Message.keyword": "/Network/ConnectPeer"
				  }
				},
				{
				  "match": {
					"Message.keyword": "/Network/DisconnectPeer"
				  }
				},
				{
				  "match": {
					"Message.keyword": "/ApplicationState/Startup"
				  }
				},
				{
				  "match": {
					"Message.keyword": "/ApplicationState/Shutdown"
				  }
				}
			  ],
			  "minimum_should_match": 1
			}
		  },
		  {
			"range": {
			  "@timestamp": {
				"from": "{FromDate}",
				"to": "{ToDate}",
				"include_lower": true,
				"include_upper": true,
				"boost": 1
			  }
			}
		  }
		]
	  }
	}
}`

func formatVoteSentQuery(round basics.Round, from time.Time, to time.Time) string {
	out := strings.Replace(voteQueryTemplate, "{Round}", fmt.Sprintf("%d", round), 1)
	out = strings.Replace(out, "{FromDate}", from.Format(time.RFC3339), 1)
	out = strings.Replace(out, "{ToDate}", to.Format(time.RFC3339), 1)
	return out
}

func formatConnectionQuery(from time.Time, to time.Time) string {
	out := strings.Replace(connectionQueryTemplate, "{FromDate}", from.Format(time.RFC3339), 1)
	out = strings.Replace(out, "{ToDate}", to.Format(time.RFC3339), 1)
	return out
}

type voteSentEventData struct {
	Sender        string // i.e. Host
	Timestamp     time.Time
	Authenticator string // i.e. Address
	Round         basics.Round
	Period        int
	Step          int
	Weight        int
}

type esDoc struct {
	Hits esHits `json:"hits"`
}
type esHits struct {
	Hits []esHitElement `json:"hits"`
}
type esHitElement struct {
	Source esSource `json:"_source"`
}
type esSource struct {
	Host      string `json:"Host"`
	Timestamp string `json:"@timestamp"`
	Data      esData `json:"Data"`
	Message   string `json:"Message"`
}
type esData struct {
	Details esDetails `json:"details"`
}
type esDetails struct {
	Address      string `json:"Address"`
	Round        int    `json:"Round"`
	Period       int    `json:"Period"`
	Step         int    `json:"Step"`
	Weight       int    `json:"Weight"`
	HostName     string `json:"HostName"`
	Incoming     bool   `json:"Incoming"`
	InstanceName string `json:"InstanceName"`
}

func queryVotes(round basics.Round, from time.Time, to time.Time) (votes []voteSentEventData) {
	if from.IsZero() {
		from = time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	if to.IsZero() {
		to = time.Now()
	}
	queryString := formatVoteSentQuery(round, from, to)

	client := &http.Client{}
	req, err := http.NewRequest("POST", "https://1ae9f9654b25441090fe5c48c833b95a.us-east-1.aws.found.io:9243/stable-mainnet-v1.0/_search", strings.NewReader(queryString))
	req.SetBasicAuth("elastic", "JUO4Mn33qd6mFYC8aQA6MLQb")
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Unable to retrieve votes from elastic search : %v \n", err)
		return
	}
	bodyText, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Unable to read votes from elastic search : %v \n", err)
		return
	}

	var doc esDoc
	err = json.Unmarshal(bodyText, &doc)
	if err != nil {
		fmt.Printf("Unable to parse votes from elastic search : %v \n", err)
		return
	}
	for _, hit := range doc.Hits.Hits {
		src := hit.Source
		var vote voteSentEventData
		vote.Sender = src.Host
		timestamp, err := time.Parse(time.RFC3339, src.Timestamp)
		if err != nil {
			fmt.Printf("Unable to parse timestamp %s : %v\n", src.Timestamp, err)
			continue
		}
		vote.Timestamp = timestamp
		vote.Authenticator = src.Data.Details.Address
		if len(vote.Authenticator) == 52 {
			// entries are missing checksum; hack around it.
			decoded, _ := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(vote.Authenticator)
			shortAddressHash := crypto.Hash(decoded[:])
			checksum := shortAddressHash[len(shortAddressHash)-4:]
			decodedAddress := append(decoded[:], checksum[:]...)
			reencoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(decodedAddress)

			vote.Authenticator = reencoded
		}
		vote.Round = basics.Round(src.Data.Details.Round)
		vote.Period = src.Data.Details.Period
		vote.Step = src.Data.Details.Step
		vote.Weight = src.Data.Details.Weight
		votes = append(votes, vote)
	}
	return
}

type esNode struct {
	guid    string
	address string
	name    string
	conn    map[*esNode]bool
}
type esConnections struct {
	nodes    []*esNode
	guidNode map[string]*esNode
}

func isValidNodeName(host string) (valid bool, guid string) {
	if len(host) < 36 {
		return
	}
	splitted := strings.Split(host, ":")
	if len(splitted) == 0 {
		return
	}
	if len(splitted[0]) != 36 {
		return
	}

	return true, splitted[0]
}

func queryConnections(from time.Time, to time.Time, initialConnection *esConnections) (err error) {
	if from.IsZero() {
		from = time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	if to.IsZero() {
		to = time.Now()
	}
	if initialConnection == nil {
		initialConnection = &esConnections{}
	}

	if initialConnection.guidNode == nil {
		initialConnection.guidNode = make(map[string]*esNode)
	}
	queryString := formatConnectionQuery(from, to)
	client := &http.Client{}
	req, err := http.NewRequest("POST", "https://1ae9f9654b25441090fe5c48c833b95a.us-east-1.aws.found.io:9243/stable-mainnet-v1.0/_search", strings.NewReader(queryString))
	req.SetBasicAuth("elastic", "JUO4Mn33qd6mFYC8aQA6MLQb")
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Unable to retrieve connections from elastic search : %v \n", err)
		return err
	}
	bodyText, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Unable to read connections from elastic search : %v \n", err)
		return err
	}

	var doc esDoc
	err = json.Unmarshal(bodyText, &doc)
	if err != nil {
		fmt.Printf("Unable to parse connections from elastic search : %v \n", err)
		return err
	}
	for _, hit := range doc.Hits.Hits {
		src := hit.Source
		var srcNode, otherNode *esNode
		var hasSrcNode, hasOtherNode bool
		srcHostValid, srcGUID := isValidNodeName(src.Host)
		if !srcHostValid {
			continue
		}
		if src.Message == "/ApplicationState/Startup" || src.Message == "/ApplicationState/Shutdown" {
			srcNode, hasSrcNode = initialConnection.guidNode[srcGUID]
			if !hasSrcNode {
				// node that we don't know about just started. ignore for now.
				continue
			}
			// we need to disconnect this node, both ways :
			for dest := range srcNode.conn {
				delete(dest.conn, srcNode)
			}
			srcNode.conn = make(map[*esNode]bool)
			continue
		}
		otherHostValid, otherGUID := isValidNodeName(src.Data.Details.HostName)
		if !otherHostValid {
			continue
		}

		if src.Message == "/Network/ConnectPeer" || src.Message == "/Network/DisconnectPeer" {
			// make sure we have both entries.
			srcNode, hasSrcNode = initialConnection.guidNode[srcGUID]
			if !hasSrcNode {
				// add
				srcNode = &esNode{
					guid: srcGUID,
					name: src.Host,
					conn: make(map[*esNode]bool),
				}
				initialConnection.nodes = append(initialConnection.nodes, srcNode)
				initialConnection.guidNode[srcGUID] = srcNode
			} else {
				srcNode.name = src.Host
			}

			otherNode, hasOtherNode = initialConnection.guidNode[otherGUID]
			if !hasOtherNode {
				// add
				otherNode = &esNode{
					guid:    otherGUID,
					name:    src.Data.Details.HostName,
					address: src.Data.Details.Address,
					conn:    make(map[*esNode]bool),
				}
				initialConnection.nodes = append(initialConnection.nodes, otherNode)
				initialConnection.guidNode[otherGUID] = otherNode
			} else {
				otherNode.address = src.Data.Details.Address
			}
		}
		if src.Message == "/Network/ConnectPeer" {
			srcNode.conn[otherNode] = true
			otherNode.conn[srcNode] = true
			//fmt.Printf("Connection : %s <---> %s\n", src.Host, src.Data.Details.HostName)
		} else if src.Message == "/Network/DisconnectPeer" {
			if srcNode.conn[otherNode] {
				delete(srcNode.conn, otherNode)
			}
			if otherNode.conn[srcNode] {
				delete(otherNode.conn, srcNode)
			}
			//fmt.Printf("Disconnection : %s <-/-> %s\n", src.Host, src.Data.Details.HostName)
		}
	}
	/*for _, node := range initialConnection.nodes {
		fmt.Printf("node %s (%s) has %d connections.\n", node.guid, node.address, len(node.conn))
	}*/
	return nil
}
