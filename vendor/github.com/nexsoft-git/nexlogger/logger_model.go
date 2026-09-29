package nexlogger

import "os"

var (
	resource    string
	application string
	version     string
)

type LoggerModel struct {
	AccessToken string `json:"access_token"`
	ClientID    string `json:"client_id"`
	UserID      string `json:"user_id" `
	Category    string `json:"category"`

	Resource    string `json:"resource"`
	Version     string `json:"version"`
	Application string `json:"application"`
	IP          string `json:"ip"`
	PID         int    `json:"pid"`
	Thread      string `json:"thread"`

	RequestID string `json:"request_id"`
	Source    string `json:"source"`

	ProcessingTime int64  `json:"processing_time"`
	ByteIn         int    `json:"byte_in"`
	ByteOut        int    `json:"byte_out"`
	Status         int    `json:"status" `
	Code           string `json:"code"`
}

func InitLoggerModel(
	resourceName,
	applicationName,
	currentVersion string,
) {
	resource = resourceName
	application = applicationName
	version = currentVersion
}

func NewloggerModel() (
	output LoggerModel,
) {
	output.IP = "-"
	output.Category = "-"
	output.PID = os.Getpid()
	output.Thread = "-"
	output.RequestID = "-"
	output.Source = "-"
	output.AccessToken = "-"
	output.Resource = resource
	output.Application = application
	output.Version = version
	output.Code = "-"
	output.ClientID = "-"
	output.UserID = "-"

	return
}
