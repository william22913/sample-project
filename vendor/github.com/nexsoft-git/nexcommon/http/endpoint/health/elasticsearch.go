package health

import (
	"context"
	"time"

	"github.com/nexsoft-git/nexcommon/util"
)

type elasticConn struct {
	elastic util.ElasticSearchUtil
}

func (e *elasticConn) Ping() string {
	status := "UP"
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if _, err := e.elastic.GetClient().ClusterHealth().Do(ctx); err != nil {
		status = "DOWN"
	}

	return status
}

func NewElasticConnChecker(
	elastic util.ElasticSearchUtil,
) *elasticConn {
	return &elasticConn{
		elastic: elastic,
	}
}
