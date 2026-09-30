package idempotency

import (
	"sync"
	"time"

	"github.com/nexsoft-git/nexcommon/model"
	"github.com/nexsoft-git/nexlogger/log"
)

func NewDatabaseBasedStorageIdempotency(
	dao PgIdempotencyDAO,
) IdempotencyProcessing {
	result := &DBBasedStorageIdempotency{
		dao: dao,
	}

	go result.lookUpExpiredIdempotencyProcess()

	return result
}

type DBBasedStorageIdempotency struct {
	sync.Mutex

	state chan (struct{})
	dao   PgIdempotencyDAO
}

func (r *DBBasedStorageIdempotency) Close() {
	r.state <- struct{}{}
}

func (r *DBBasedStorageIdempotency) AddProcessingService(
	path string,
	idempotencyKey string,
) (
	result *ServiceProcess,
	err error,
) {
	r.Lock()
	defer r.Unlock()

	key := ToIdempotencyKey(path, idempotencyKey)

	tx, idempotencyModel, err := r.dao.CheckIdempotency(key)

	if err != nil {
		return
	}

	result = &ServiceProcess{
		Tx: tx,
		result: model.IdempotencyServiceProcessResult{
			Header: idempotencyModel.Result.Header,
			Output: idempotencyModel.Result.Output,
			Type:   idempotencyModel.Result.Type,
		},
		timeStored: idempotencyModel.CreatedAt.Time,
		Func:       r.dao.SetIdempotencyResult,
	}

	return

}

func (i *DBBasedStorageIdempotency) lookUpExpiredIdempotencyProcess() {
	ticker := time.NewTicker(IDEMPOTENCY_CLEANUP_CHECKER_TIME)
	i.state = make(chan struct{})

	for {
		select {
		case <-ticker.C:
			err := i.dao.DeleteExpiredIdempotencyKey(IDEMPOTENCY_CLEANUP_DURATION)
			if err != nil {
				log.Error().
					Err(err).
					Caller().
					Msg("Error Found When delete expired Idempotency Key")
			}
		case <-i.state:
			return
		}
	}

}
