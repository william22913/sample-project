// Package dao holds the SQL. One file per entity, one struct per table, and
// no business rules - a DAO answers "what does the database say", never "is
// this allowed". Mirrors kelasku's dao/ package.
package dao

import (
	"context"
	"time"
)

// queryTimeout bounds a read's worst-case latency, so a slow or blocked query
// (lock contention, failover, network blip) cannot hold a serving goroutine -
// and transitively a connection out of the shared pool - indefinitely.
//
// Writes do not get this wrapper. They run inside a transaction the caller
// owns and already bounds, and cancelling a context mid-transaction would turn
// a slow write into a rollback.
const queryTimeout = 5 * time.Second

func withQueryTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, queryTimeout)
}
