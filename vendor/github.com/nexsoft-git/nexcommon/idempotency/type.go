package idempotency

import (
	"fmt"
	"time"
)

var IDEMPOTENCY_CLEANUP_DURATION = time.Duration(10 * time.Minute)
var IDEMPOTENCY_CLEANUP_CHECKER_TIME = time.Duration(1 * time.Minute)

func ToIdempotencyKey(
	path string,
	idempotencyKey string,
) string {
	return fmt.Sprintf("%s->%s", path, idempotencyKey)
}

type IdempotencyProcessing interface {
	AddProcessingService(
		path string,
		idempotencyKey string,
	) (
		result *ServiceProcess,
		err error,
	)

	Close()
}
