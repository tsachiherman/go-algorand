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

// Package db defines database utility functions.
//
// These functions currently work on a sqlite database.
// Other databases may not work with functions in this package.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/mattn/go-sqlite3"

	"github.com/algorand/go-algorand/logging"
)

/* database utils */

// busy is the time to wait for a sqlite lock from another process, in ms.
// This causes sqlite to wait before returning SQLITE_BUSY.  On the other
// hand, conflicts with other connections from the same process (e.g., algod)
// might contend on the shared cache, which corresponds to SQLITE_LOCKED and
// is not covered by the busy timeout.  We rely on sqlite_unlock_notify()
// to wait for the shared cache lock to be released.  This is enabled in
// go-sqlite3 with the "sqlite_unlock_notify" Go build tag.
const busy = 1000

// initStatements is a list of statements we execute after opening a database
// connection.  For now, it's just an optional "PRAGMA fullfsync=true" on
// MacOSX.
var initStatements []string
var sqliteInitOnce sync.Once

// An Accessor manages a sqlite database handle and any outstanding batching operations.
type Accessor struct {
	Handle   *sql.DB
	readOnly bool
	inMemory bool
	log      logging.Logger
}

// VacuumStats returns the database statistics before and after a vacuum operation
type VacuumStats struct {
	// PagesBefore is the number of pages in the database before the vacuum operation
	PagesBefore uint64
	// SizeBefore is the amount of data used by the database ( number of pages * size of a page) before the vacuum operation
	SizeBefore uint64
	// PagesAfter is the number of pages in the database after the vacuum operation
	PagesAfter uint64
	// SizeAfter is the amount of data used by the database ( number of pages * size of a page) after the vacuum operation
	SizeAfter uint64
}

// WALCheckpointMode defines the various checkpoint modes supported by SQLlite
type WALCheckpointMode uint32

const (
	// WALCheckpointModePassive checkpoint as many frames as possible without waiting for any database readers or writers to finish. Sync the db file if all frames in the log are checkpointed.
	WALCheckpointModePassive = WALCheckpointMode(0)
	// WALCheckpointModeFull This mode blocks (invokes the busy-handler callback) until there is no database writer and all readers are reading from the most recent database snapshot. It then checkpoints all frames in the log file and syncs the database file. FULL blocks concurrent writers while it is running, but readers can proceed.
	WALCheckpointModeFull = WALCheckpointMode(1)
	// WALCheckpointModeRestart This mode works the same way as FULL with the addition that after checkpointing the log file it blocks (calls the busy-handler callback) until all readers are finished with the log file. This ensures that the next client to write to the database file restarts the log file from the beginning. RESTART blocks concurrent writers while it is running, but allowed readers to proceed.
	WALCheckpointModeRestart = WALCheckpointMode(2)
	// WALCheckpointModeTruncate This mode works the same way as RESTART with the addition that the WAL file is truncated to zero bytes upon successful completion.
	WALCheckpointModeTruncate = WALCheckpointMode(3)
)

// txExecutionContext contains the data that is associated with every created transaction
// before sending it to the user-defined callback. This allows the callback function to
// make changes to the execution setting of an ongoing transaction.
type txExecutionContext struct {
	deadline time.Time
}

// MakeAccessor creates a new Accessor.
func MakeAccessor(dbfilename string, readOnly bool, inMemory bool) (Accessor, error) {
	return makeAccessorImpl(dbfilename, readOnly, inMemory, []string{"_journal_mode=wal"})
}

// MakeErasableAccessor creates a new Accessor with the secure_delete pragma set;
// see https://www.sqlite.org/pragma.html#pragma_secure_delete
// It is not read-only and not in-memory (otherwise, erasability doesn't matter)
func MakeErasableAccessor(dbfilename string) (Accessor, error) {
	return makeAccessorImpl(dbfilename, false, false, []string{"_secure_delete=on"})
}

func makeAccessorImpl(dbfilename string, readOnly bool, inMemory bool, params []string) (Accessor, error) {
	var db Accessor
	db.readOnly = readOnly
	db.inMemory = inMemory

	// SQLite3 driver we use (mattn/go-sqlite3) does not implement driver.DriverContext interface
	// that forces sql.Open calling sql.OpenDB and return a struct without any touches to the underlying driver.
	// Because of that SQLite library is not initialized until the very first call of sqlite3_open_v2 that happens
	// in sql.DB.conn. SQLite initialization is not thread-safe on date of writing (2/27/2020) and
	// mattn/go-sqlite3 has no special code to handle this case.
	// Solution is to create a connection using a safe synchronization barrier right here.
	// The connection goes to a connection pool inside Go's sql package and will be re-used when needed.
	// See https://github.com/algorand/go-algorand/issues/846 for more details.
	var err error
	db.Handle, err = sql.Open("sqlite3", URI(dbfilename, readOnly, inMemory)+"&"+strings.Join(params, "&"))

	if err == nil {
		// create a connection to safely initialize SQLite once
		initFn := func() {
			var conn *sql.Conn
			if conn, err = db.Handle.Conn(context.Background()); err != nil {
				db.Close()
				return
			}
			if err = conn.Close(); err != nil {
				db.Close()
			}
		}
		sqliteInitOnce.Do(initFn)
		if err != nil {
			// init failed, db closed and err is set
			return db, err
		}
		err = db.runInitStatements()
	}

	return db, err
}

// runInitStatements executes initialization statements.
func (db *Accessor) runInitStatements() error {
	for _, stmt := range initStatements {
		_, err := db.Handle.Exec(stmt)
		if err != nil {
			db.Close()
			return err
		}
	}

	return nil
}

// SetLogger sets the Logger, mainly for unit test quietness
func (db *Accessor) SetLogger(log logging.Logger) {
	db.log = log
}

func (db *Accessor) logger() logging.Logger {
	if db.log != nil {
		return db.log
	}
	return logging.Base()
}

// Close closes the connection.
func (db *Accessor) Close() {
	db.Handle.Close()
	db.Handle = nil
}

// LoggedRetry executes a function repeatedly as long as it returns an error
// that indicates database contention that warrants a retry.
// Sends warnings and errors to log.
func LoggedRetry(fn func() error, log logging.Logger) (err error) {
	for i := 0; (i == 0) || dbretry(err); i++ {
		if i > 0 {
			if i < infoTxRetries {
				log.Infof("db.LoggedRetry: %d retries (last err: %v)", i, err)
			} else if i >= 1000 {
				log.Errorf("db.LoggedRetry: %d retries (last err: %v)", i, err)
				return
			} else if i%warnTxRetriesInterval == 0 {
				log.Warnf("db.LoggedRetry: %d retries (last err: %v)", i, err)
			}
		}
		err = fn()
	}
	return
}

// Retry executes a function repeatedly as long as it returns an error
// that indicates database contention that warrants a retry.
// Sends warnings and errors to logging.Base()
func Retry(fn func() error) (err error) {
	return LoggedRetry(fn, logging.Base())
}

// getDecoratedLogger retruns a decorated logger that includes the readonly true/false, caller and extra fields.
func (db *Accessor) getDecoratedLogger(fn idemFn, extras ...interface{}) logging.Logger {
	log := db.logger().With("readonly", db.readOnly)
	_, file, line, ok := runtime.Caller(3)
	if ok {
		log = log.With("caller", fmt.Sprintf("%s:%d", file, line))
	}
	log = log.With("callee", runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name())
	for i, e := range extras {
		if e == nil || reflect.ValueOf(e).IsNil() {
			continue
		}
		log = log.With(fmt.Sprintf("extra(%d)", i), e)
	}

	return log
}

// IsSharedCacheConnection returns whether this connection was created using shared-cache connection or not.
// we use shared cache for in-memory databases
func (db *Accessor) IsSharedCacheConnection() bool {
	return db.inMemory
}

// Atomic executes a piece of code with respect to the database atomically.
// For transactions where readOnly is false, sync determines whether or not to wait for the result.
func (db *Accessor) Atomic(fn idemFn, extras ...interface{}) (err error) {
	return db.atomic(fn, nil, extras...)
}

// Atomic executes a piece of code with respect to the database atomically.
// For transactions where readOnly is false, sync determines whether or not to wait for the result.
func (db *Accessor) atomic(fn idemFn, commitLocker sync.Locker, extras ...interface{}) (err error) {
	atomicDeadline := time.Now().Add(time.Second)

	// note that the sql library will drop panics inside an active transaction
	guardedFn := func(ctx context.Context, tx *sql.Tx) (err error) {
		defer func() {
			if r := recover(); r != nil {
				var ok bool
				err, ok = r.(error)
				if !ok {
					err = fmt.Errorf("%v", r)
				}
			}
		}()

		err = fn(ctx, tx)
		return
	}

	var tx *sql.Tx
	var conn *sql.Conn
	ctx := context.Background()

	commitWriteLockTaken := false
	if commitLocker != nil && db.IsSharedCacheConnection() {
		// When we're using in memory database, the sqlite implementation forces us to use a shared cache
		// mode so that multiple connections ( i.e. read and write ) could share the database instance.
		// ( it would also create issues between precompiled statements and regular atomic calls, as the former
		// would generate a connection on the fly).
		// when using a shared cache, we have to be aware that there are additional locking mechanisms that are
		// internal to the sqlite. Two of them which play a role here are the sqlite_unlock_notify which
		// prevents a shared cache locks from returning "database is busy" error and would block instead, and
		// table level locks, which ensure that at any one time, a single table may have any number of active
		// read-locks or a single active write lock.
		// see https://www.sqlite.org/sharedcache.html for more details.
		// These shared cache constrains are more strict than the WAL based concurrency limitations, which allows
		// one writer and multiple readers at the same time.
		// In particular, the shared cache limitation means that since a connection could become a writer, any syncronization
		// operating that would prevent this operation from completing could result with a deadlock.
		// This is the reason why for shared cache connections, we'll take the lock before starting the write transaction,
		// and would keep it along. It will cause a degraded performance when using a shared cache connection
		// compared to a private cache connection, but would grentee correct locking semantics.
		commitLocker.Lock()
		commitWriteLockTaken = true
	}

	for i := 0; (i == 0) || dbretry(err); i++ {
		if i > 0 {
			if i < infoTxRetries {
				db.getDecoratedLogger(fn, extras).Infof("db.atomic: %d connection retries (last err: %v)", i, err)
			} else if i >= 1000 {
				db.getDecoratedLogger(fn, extras).Errorf("db.atomic: %d connection retries (last err: %v)", i, err)
				break
			} else if i%warnTxRetriesInterval == 0 {
				db.getDecoratedLogger(fn, extras).Warnf("db.atomic: %d connection retries (last err: %v)", i, err)
			}
		}
		conn, err = db.Handle.Conn(ctx)
	}

	if err != nil {
		// fail case - unable to create database connection
		if commitLocker != nil && commitWriteLockTaken {
			commitLocker.Unlock()
		}
		return
	}
	defer conn.Close()

	for i := 0; ; i++ {
		// check if the lock was taken in previous iteration
		if commitLocker != nil && (!db.IsSharedCacheConnection()) && commitWriteLockTaken {
			// undo the lock.
			commitLocker.Unlock()
			commitWriteLockTaken = false
		}

		if i > 0 {
			if i < infoTxRetries {
				db.getDecoratedLogger(fn, extras).Infof("db.atomic: %d retries (last err: %v)", i, err)
			} else if i >= 1000 {
				db.getDecoratedLogger(fn, extras).Errorf("db.atomic: %d retries (last err: %v)", i, err)
				break
			} else if i%warnTxRetriesInterval == 0 {
				db.getDecoratedLogger(fn, extras).Warnf("db.atomic: %d retries (last err: %v)", i, err)
			}
		}

		tx, err = conn.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable, ReadOnly: db.readOnly})
		if dbretry(err) {
			continue
		} else if err != nil {
			break
		}

		// create a transaction context data
		txContextData := &txExecutionContext{
			deadline: atomicDeadline,
		}

		err = guardedFn(context.WithValue(ctx, tx, txContextData), tx)
		if err != nil {
			tx.Rollback()
			if dbretry(err) {
				continue
			} else {
				break
			}
		}

		// if everytyhing went well, take the lock, as we're going to attempt to commit the transaction to database.
		if commitLocker != nil && (!commitWriteLockTaken) && (!db.IsSharedCacheConnection()) {
			commitLocker.Lock()
			commitWriteLockTaken = true
		}

		err = tx.Commit()
		if err == nil {
			// update the deadline, as it might have been updated.
			atomicDeadline = txContextData.deadline
			break
		} else if !dbretry(err) {
			break
		}
	}

	// if we've errored, make sure to unlock the commitLocker ( if there is any )
	if err != nil && commitLocker != nil && commitWriteLockTaken {
		commitLocker.Unlock()
	}

	if time.Now().After(atomicDeadline) {
		db.getDecoratedLogger(fn, extras).Warnf("dbatomic: tx surpassed expected deadline by %v", time.Now().Sub(atomicDeadline))
	}
	return
}

// ResetTransactionWarnDeadline allow the atomic function to extend it's warn deadline by setting a new deadline.
// The Accessor can be copied and therefore isn't suitable for multi-threading directly,
// however, the transaction context and transaction object can be used to uniquely associate the request
// with a particular deadline.
// the function fails if the given transaction is not on the stack of the provided context.
func ResetTransactionWarnDeadline(ctx context.Context, tx *sql.Tx, deadline time.Time) (prevDeadline time.Time, err error) {
	txContextData, ok := ctx.Value(tx).(*txExecutionContext)
	if !ok {
		// it's not a valid call. just return an error.
		return time.Time{}, fmt.Errorf("the provided tx does not have a valid txExecutionContext object in it's context")
	}
	prevDeadline = txContextData.deadline
	txContextData.deadline = deadline
	return
}

// AtomicCommitWriteLock executes a piece of code with respect to the database atomically.
// For transactions where readOnly is false, sync determines whether or not to wait for the result.
// The commitLocker is being taken before the transaction is committed. In case of an error, the lock would get released.
// on all success cases ( i.e. err = nil ) the lock would be taken. on all the fail cases, the lock would be released
func (db *Accessor) AtomicCommitWriteLock(fn idemFn, commitLocker sync.Locker, extras ...interface{}) (err error) {
	return db.atomic(fn, commitLocker, extras...)
}

// Vacuum perform a full-vacuum on the given database. In order for the vacuum to succeed, the storage needs to have
// double the amount of the current database size ( roughly ), and we cannot have any other transaction ( either read
// or write ) being active.
func (db *Accessor) Vacuum(ctx context.Context) (stats VacuumStats, err error) {
	if db.readOnly {
		return stats, fmt.Errorf("read-only database was used to attempt and perform vacuuming")
	}
	if db.inMemory {
		return stats, nil
	}
	pageSize, err2 := db.GetPageSize(ctx)
	if err2 != nil {
		return stats, err2
	}
	stats.PagesBefore, err = db.GetPageCount(ctx)
	if err != nil {
		return stats, err
	}
	stats.SizeBefore = pageSize * stats.PagesBefore
	_, err = db.Handle.ExecContext(ctx, "VACUUM")
	if err != nil {
		return stats, err
	}
	stats.PagesAfter, err = db.GetPageCount(ctx)
	if err != nil {
		return stats, err
	}
	stats.SizeAfter = pageSize * stats.PagesAfter
	return
}

// URI returns the sqlite URI given a db filename as an input.
func URI(filename string, readOnly bool, memory bool) string {
	uri := fmt.Sprintf("file:%s?_busy_timeout=%d&_synchronous=full", filename, busy)
	if !readOnly {
		uri += "&_txlock=immediate"
	}
	if memory {
		uri += "&mode=memory"
		uri += "&cache=shared"
	}
	return uri
}

// GetPageCount returns the total number of pages in the database
func (db *Accessor) GetPageCount(ctx context.Context) (pageCount uint64, err error) {
	err = db.Handle.QueryRowContext(ctx, "PRAGMA page_count").Scan(&pageCount)
	if err == sql.ErrNoRows {
		err = fmt.Errorf("sqlite database doesn't support `PRAGMA page_count`")
	}
	return
}

// GetPageSize returns the number of bytes per database page
func (db *Accessor) GetPageSize(ctx context.Context) (pageSize uint64, err error) {
	err = db.Handle.QueryRowContext(ctx, "PRAGMA page_size").Scan(&pageSize)
	if err == sql.ErrNoRows {
		err = fmt.Errorf("sqlite database doesn't support `PRAGMA page_size`")
	}
	return
}

// WALCheckpoint invokes a checkpoint on the WAL file using the requested mode
func WALCheckpoint(ctx context.Context, tx *sql.Tx, mod WALCheckpointMode) (err error) {
	checkpointModeCmd := map[WALCheckpointMode]string{
		WALCheckpointModePassive:  "PRAGMA wal_checkpoint(PASSIVE)",
		WALCheckpointModeFull:     "PRAGMA wal_checkpoint(FULL)",
		WALCheckpointModeRestart:  "PRAGMA wal_checkpoint(RESTART)",
		WALCheckpointModeTruncate: "PRAGMA wal_checkpoint(TRUNCATE)",
	}
	cmd, ok := checkpointModeCmd[mod]
	if !ok {
		return fmt.Errorf("invalid WALCheckpointMode specified : %v", mod)
	}
	_, err = tx.ExecContext(ctx, cmd)
	return
}

// dbretry returns true if the error might be temporary
func dbretry(obj error) bool {
	err, ok := obj.(sqlite3.Error)
	return ok && (err.Code == sqlite3.ErrLocked || err.Code == sqlite3.ErrBusy)
}

type idemFn func(ctx context.Context, tx *sql.Tx) error

const infoTxRetries = 5
const warnTxRetriesInterval = 1
