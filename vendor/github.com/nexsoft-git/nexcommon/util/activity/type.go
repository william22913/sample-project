package activity

import (
	"github.com/nats-io/nats.go"
	"github.com/nexsoft-git/nexcommon/model"
)

type Publisher struct {
	Nats    nats.JetStreamContext
	Subject string
}

type ActivityUtil interface {
	PushActivity(activity model.ActivityDTO)
	Close()
	EnableLogAllActivity()
	IsLogAllActivity() bool
}
