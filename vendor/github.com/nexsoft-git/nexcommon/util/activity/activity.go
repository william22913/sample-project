package activity

import (
	"encoding/json"

	"github.com/nexsoft-git/nexcommon/model"
	"github.com/nexsoft-git/nexlogger/log"
)

type activityUtil struct {
	state            chan (struct{})
	channel          chan (model.ActivityDTO)
	isLogAllActivity bool
	publisher        Publisher
}

func NewActivityService(
	publisher Publisher,
) ActivityUtil {
	service := activityUtil{
		state:     make(chan (struct{})),
		channel:   make(chan (model.ActivityDTO)),
		publisher: publisher,
	}

	go service.consumeActivity()

	return &service
}

func (a *activityUtil) EnableLogAllActivity() {
	a.isLogAllActivity = true
}

func (a *activityUtil) IsLogAllActivity() bool {
	return a.isLogAllActivity
}

func (a *activityUtil) PushActivity(
	activity model.ActivityDTO,
) {
	go func() {
		a.channel <- activity
	}()
}

func (a *activityUtil) Close() {
	a.state <- struct{}{}
}

func (a *activityUtil) consumeActivity() {
	for {
		select {
		case activity := <-a.channel:

			data, _ := json.Marshal(activity)
			_, err := a.publisher.Nats.Publish(a.publisher.Subject, data)

			if err != nil {
				log.Error().
					Err(err).
					Caller().
					Msg("Error Found when publish acivity message")
			}

		case <-a.state:
			return
		}
	}
}
