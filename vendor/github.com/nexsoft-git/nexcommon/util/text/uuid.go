package text

import (
	"strings"

	"github.com/google/uuid"
)

func GetUUID() (output string) {
	UUID, _ := uuid.NewRandom()
	output = UUID.String()
	output = strings.Replace(output, "-", "", -1)
	return
}
