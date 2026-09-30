package idempotency

import (
	"database/sql"
	"sync"
	"time"

	"github.com/nexsoft-git/nexcommon/model"
)

type ServiceProcess struct {
	sync.Mutex
	Tx   *sql.Tx
	Func func(
		idempotencyModel model.IdempotencyModel,
	) error
	result     model.IdempotencyServiceProcessResult
	timeStored time.Time
}

func (s *ServiceProcess) GetServiceProcessResult() model.IdempotencyServiceProcessResult {
	s.Lock()
	return s.result
}

func (s *ServiceProcess) Close() {
	s.Unlock()
}

func (s *ServiceProcess) StopProcess(
	result model.IdempotencyServiceProcessResult,
) {
	s.result = result
	s.Unlock()
}
