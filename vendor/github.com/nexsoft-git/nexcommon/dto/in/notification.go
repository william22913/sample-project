package in

type NotificationMessagingRequest struct {
	Client  string      `json:"client" required:"client"`
	Topic   string      `json:"topic" required:"topic,client" empty:"allowed"`
	Message interface{} `json:"message" required:"topic,client"`
}
