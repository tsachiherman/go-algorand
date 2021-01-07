package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/algorand/go-algorand/crypto"
	"github.com/algorand/go-algorand/data/basics"
	"github.com/algorand/go-algorand/protocol"

	//badger "github.com/dgraph-io/badger/v2"
	bolt "go.etcd.io/bbolt"
)

var (
	accountbase []byte = []byte("a")
)

func main() {
	//db, err := badger.Open(badger.DefaultOptions("/tmp/badger"))
	dbpath := "/tmp/wat.db"
	os.Remove(dbpath)
	db, err := bolt.Open(dbpath, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(accountbase)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	makeAccountStart := time.Now()
	totalStartupAccountsNumber := 5000000
	batchCount := 1000
	accountsAddress := make([][]byte, 0, totalStartupAccountsNumber)
	accountBytes := uint64(0)
	for batch := 0; batch <= batchCount; batch++ {
		fmt.Printf("\033[M\r %d / %d accounts written", totalStartupAccountsNumber*batch/batchCount, totalStartupAccountsNumber)
		acctsData := generateRandomTestingAccountBalances(totalStartupAccountsNumber / batchCount)
		err = db.Update(func(tx *bolt.Tx) error {
			b := tx.Bucket(accountbase)
			for addr, acctData := range acctsData {
				acctd := protocol.Encode(&acctData)
				accountBytes += uint64(len(acctd))
				err = b.Put(addr[:], acctd)
				if err != nil {
					return err
				}
				accountsAddress = append(accountsAddress, addr[:])
			}
			return nil
		})
		if err != nil {
			log.Fatal(err)
		}
	}
	makeAccountEnd := time.Now()
	dt := makeAccountEnd.Sub(makeAccountStart)
	fmt.Printf("\033[M\r")
	fmt.Printf("%d accounts written to new db in %s, %f ns/acct_insert\n", len(accountsAddress), dt.String(), float64(dt.Nanoseconds())/float64(len(accountsAddress)))
	fmt.Printf("%d bytes/acct\n", accountBytes/uint64(len(accountsAddress)))

	updateOrder := rand.Perm(len(accountsAddress))
	randomAccountData := make([]byte, 500)
	crypto.RandBytes(randomAccountData)

	updateStart := time.Now()
	upcount := 0
	roundBatch := 20000
	pos := 0
	for {
		crypto.RandBytes(randomAccountData)
		err = db.Update(func(tx *bolt.Tx) error {
			b := tx.Bucket(accountbase)
			for i := 0; i < roundBatch; i++ {
				addr := accountsAddress[updateOrder[pos]]
				err = b.Put(addr, randomAccountData)
				if err != nil {
					return err
				}
				pos = (pos + 1) % len(updateOrder)
			}
			return nil
		})
		if err != nil {
			log.Fatal(err)
		}
		upcount += roundBatch
		t := time.Now()
		dt = t.Sub(updateStart)
		if dt > 5*time.Second {
			fmt.Printf("%d accounts updated in %s, %f ns/acct_update\n", upcount, dt.String(), float64(dt.Nanoseconds())/float64(upcount))
			break
		}
	}
	defer db.Close()
}

func randomAddress() basics.Address {
	var addr basics.Address
	crypto.RandBytes(addr[:])
	return addr
}

func generateRandomTestingAccountBalances(numAccounts int) (updates map[basics.Address]basics.AccountData) {
	secrets := crypto.GenerateOneTimeSignatureSecrets(15, 500)
	pubVrfKey, _ := crypto.VrfKeygenFromSeed([32]byte{0, 1, 2, 3})
	updates = make(map[basics.Address]basics.AccountData, numAccounts)

	for i := 0; i < numAccounts; i++ {
		addr := randomAddress()
		updates[addr] = basics.AccountData{
			MicroAlgos:         basics.MicroAlgos{Raw: 0x000ffffffffffffff},
			Status:             basics.NotParticipating,
			RewardsBase:        uint64(i),
			RewardedMicroAlgos: basics.MicroAlgos{Raw: 0x000ffffffffffffff},
			VoteID:             secrets.OneTimeSignatureVerifier,
			SelectionID:        pubVrfKey,
			VoteFirstValid:     basics.Round(0x000ffffffffffffff),
			VoteLastValid:      basics.Round(0x000ffffffffffffff),
			VoteKeyDilution:    0x000ffffffffffffff,
			AssetParams: map[basics.AssetIndex]basics.AssetParams{
				0x000ffffffffffffff: {
					Total:         0x000ffffffffffffff,
					Decimals:      0x2ffffff,
					DefaultFrozen: true,
					UnitName:      "12345678",
					AssetName:     "12345678901234567890123456789012",
					URL:           "12345678901234567890123456789012",
					MetadataHash:  pubVrfKey,
					Manager:       addr,
					Reserve:       addr,
					Freeze:        addr,
					Clawback:      addr,
				},
			},
			Assets: map[basics.AssetIndex]basics.AssetHolding{
				0x000ffffffffffffff: {
					Amount: 0x000ffffffffffffff,
					Frozen: true,
				},
			},
		}
	}
	return
}
