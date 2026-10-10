package sqldb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"gorm.io/gorm"

	"tests-runner/util/stack"
	"tests-runner/util/util/errutil"
)

var (
	queryDurationsHist *prometheus.HistogramVec
	connCounts         *prometheus.GaugeVec
	connWaitTimes      *prometheus.GaugeVec
	sqldbStats         *prometheus.Registry

	requested int32
)

func init() {
	queryDurationsHist = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "sqldb",
			Name:      "query_durations_sec",
			Help:      "duration of SQL queries, groupped by the call site",
			Buckets:   prometheus.ExponentialBucketsRange(1e-4, 8, 64),
		},
		[]string{"db", "query", "fn"})

	connCounts = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "sqldb",
			Name:      "conns_inuse_cur",
			Help:      "the number of in-use DB connections",
		},
		[]string{"db"},
	)

	connWaitTimes = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "sqldb",
			Name:      "conn_wait_times",
			Help:      "how long users were blocked waiting for a DB connection",
		},
		[]string{"db"},
	)

	sqldbStats = prometheus.NewRegistry()
	sqldbStats.MustRegister(
		queryDurationsHist,
		connCounts,
		connWaitTimes,
	)
}

// NewSQLStatsCollector creates a Prometheus collector that reports DB-related
// statistics on the current process.
//
// NOTE: only DBs opened after this call will have monitoring and tracing enabled.
func NewSQLStatsCollector() prometheus.Collector {
	atomic.StoreInt32(&requested, 1)
	return sqldbStats
}

func withSQLMonitoring(db *gorm.DB, dbName string) *gorm.DB {
	if atomic.LoadInt32(&requested) == 0 {
		return db
	}

	db = db.Session(&gorm.Session{})
	db.ConnPool = &connPoolWithStats{
		connPool: db.Statement.ConnPool,
		counters: newDBCounters(dbName),
	}
	db.Statement.ConnPool = db.ConnPool
	return db
}

type connPoolWithStats struct {
	connPool gorm.ConnPool
	counters *dbCounters
}

var (
	_ gorm.GetDBConnector   = (*connPoolWithStats)(nil)
	_ gorm.ConnPool         = (*connPoolWithStats)(nil)
	_ gorm.ConnPoolBeginner = (*connPoolWithStats)(nil)
)

// GetDBConn returns underlying std DB.
func (c *connPoolWithStats) GetDBConn() (*sql.DB, error) {
	if dbConn, ok := c.connPool.(gorm.GetDBConnector); ok && dbConn != nil {
		return dbConn.GetDBConn()
	}
	if sqldb, ok := c.connPool.(*sql.DB); ok {
		return sqldb, nil
	}

	return nil, gorm.ErrInvalidDB
}

// PrepareContext simply calls underlying PrepareContext
func (c *connPoolWithStats) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	panic("GORM must be configured with PrepareStmt: false")
}

// ExecContext calls original GORM's ExecContext, and measures the time it takes.
func (c *connPoolWithStats) ExecContext(ctx context.Context, query string, args ...any) (result sql.Result, err error) {
	caller := getCallerOfSqldb()

	start := time.Now()
	res, err := c.connPool.ExecContext(ctx, query, args...)
	end := time.Now()

	c.reportTime(start, end, caller)
	c.updateConnStats()
	return res, err
}

// ExecContext calls original GORM's QueryContext, and measures the time it takes.
func (c *connPoolWithStats) QueryContext(ctx context.Context, query string, args ...any) (result *sql.Rows, err error) {
	caller := getCallerOfSqldb()

	start := time.Now()
	res, err := c.connPool.QueryContext(ctx, query, args...)
	end := time.Now()

	c.reportTime(start, end, caller)
	c.updateConnStats()
	return res, err
}

// ExecContext calls original GORM's QueryRowContext, and measures the time it takes.
func (c *connPoolWithStats) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	panic("QueryRowContext() must not be called")
}

// BeginTx starts new transaction, and reports some stats about it to Prometheus.
func (c *connPoolWithStats) BeginTx(ctx context.Context, opts *sql.TxOptions) (gorm.ConnPool, error) {
	var connPool gorm.ConnPool
	var err error

	if beginner, ok := c.connPool.(gorm.TxBeginner); ok {
		connPool, err = beginner.BeginTx(ctx, opts)
	} else if beginner, ok := c.connPool.(gorm.ConnPoolBeginner); ok {
		connPool, err = beginner.BeginTx(ctx, opts)
	} else {
		err = gorm.ErrInvalidTransaction
	}
	if err != nil {
		return nil, err
	}

	c.updateConnStats()

	return &txWithStats{
		connPoolWithStats: &connPoolWithStats{
			connPool: connPool,
			counters: c.counters,
		},
		startedAt: time.Now(),
	}, nil
}

func (c *connPoolWithStats) updateConnStats() {
	db, err := c.GetDBConn()
	if err != nil {
		// NOTE: for a transaction, c.connPool is *sql.Tx which has no interface
		// to query DBStats.
		return
	}

	s := db.Stats()

	c.counters.nrConns.Set(float64(s.InUse))
	c.counters.connWaitTime.Set(s.WaitDuration.Seconds())
}

type txWithStats struct {
	*connPoolWithStats
	startedAt time.Time
}

var (
	_ gorm.GetDBConnector = (*txWithStats)(nil)
	_ gorm.ConnPool       = (*txWithStats)(nil)
	_ gorm.TxCommitter    = (*txWithStats)(nil)
)

// Commit commits the transaction.
func (t *txWithStats) Commit() error {
	if committer, ok := t.connPool.(gorm.TxCommitter); ok {
		return committer.Commit()
	}

	return gorm.ErrInvalidTransaction
}

// Rollback rolls back the transaction.
func (t *txWithStats) Rollback() error {
	if committer, ok := t.connPool.(gorm.TxCommitter); ok {
		return committer.Rollback()
	}

	return gorm.ErrInvalidTransaction
}

func (c *connPoolWithStats) reportTime(start, end time.Time, caller stack.Frame) {
	if caller != 0 {
		c.counters.getQueryDurations(caller).Observe(end.Sub(start).Seconds())
	}
}

type dbCounters struct {
	dbName string

	lock           sync.Mutex
	queryDurations map[stack.Frame]prometheus.Observer

	nrConns      prometheus.Gauge
	connWaitTime prometheus.Gauge
}

func newDBCounters(dbName string) *dbCounters {
	nrConns, err := connCounts.GetMetricWith(prometheus.Labels{"db": dbName})
	errutil.FatalIf(err)
	connWaitTime, err := connWaitTimes.GetMetricWith(prometheus.Labels{"db": dbName})
	errutil.FatalIf(err)

	return &dbCounters{
		dbName:         dbName,
		queryDurations: make(map[stack.Frame]prometheus.Observer),
		nrConns:        nrConns,
		connWaitTime:   connWaitTime,
	}
}

func (c *dbCounters) getQueryDurations(caller stack.Frame) prometheus.Observer {
	c.lock.Lock()
	defer c.lock.Unlock()

	if counter, exists := c.queryDurations[caller]; exists {
		return counter
	}

	file, line := caller.FileLine()
	counter, err := queryDurationsHist.GetMetricWith(prometheus.Labels{
		"db":    c.dbName,
		"query": fmt.Sprintf("%s:%d", file, line),
		"fn":    caller.Name(),
	})
	errutil.FatalIf(err)

	c.queryDurations[caller] = counter
	return counter
}

func getCallerOfSqldb() stack.Frame {
	for _, caller := range stack.Callers(4 + 2) {
		name := caller.Name()
		inGORMv2 := strings.HasPrefix(name, "gorm.io")
		inSQLDB := strings.HasPrefix(name, "pkg/sqldb")
		inListok := strings.HasPrefix(name, "pkg/listok")
		// DBs of various services define InTx() that copies gorm.DB
		// and some service-specific state. Skip those, too.
		inTx := strings.HasSuffix(name, "InTx")

		if !inGORMv2 && !inSQLDB && !inListok && !inTx {
			return caller
		}
	}
	return 0
}
