package idempotency

import (
	"sync"
	"time"
)

type MemoryBasedStorageIdempotency struct {
	sync.Mutex
	processing map[string]*ServiceProcess
	state      chan (struct{})
}

func NewInMemoryBasedStorageIdempotency() IdempotencyProcessing {
	result := &MemoryBasedStorageIdempotency{
		processing: make(map[string]*ServiceProcess),
	}

	go result.lookUpExpiredIdempotencyProcess()

	return result
}

func (i *MemoryBasedStorageIdempotency) Close() {
	i.state <- struct{}{}
}

func (i *MemoryBasedStorageIdempotency) lookUpExpiredIdempotencyProcess() {
	ticker := time.NewTicker(IDEMPOTENCY_CLEANUP_CHECKER_TIME)
	i.state = make(chan struct{})

	for {
		select {
		case <-ticker.C:
			currentTime := time.Now()

			for keys := range i.processing {
				func() {
					i.Lock()
					defer i.Unlock()
					if currentTime.Sub(i.processing[keys].timeStored) >= IDEMPOTENCY_CLEANUP_DURATION {
						delete(i.processing, keys)
					}
				}()
			}

		case <-i.state:
			return
		}
	}

}

func (i *MemoryBasedStorageIdempotency) AddProcessingService(
	path string,
	idempotencyKey string,
) (
	*ServiceProcess,
	error,
) {
	i.Lock()
	defer i.Unlock()

	key := ToIdempotencyKey(path, idempotencyKey)
	if i.processing[key] == nil {
		i.processing[key] = &ServiceProcess{
			timeStored: time.Now(),
		}
	}

	return i.processing[key], nil
}
