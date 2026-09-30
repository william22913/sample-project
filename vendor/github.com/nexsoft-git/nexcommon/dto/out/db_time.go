package out

import (
	"database/sql"
	"time"

	"github.com/nexsoft-git/nexcommon/constanta"
)

type DefaultTime sql.NullTime

func (d DefaultTime) MarshalJSON() ([]byte, error) {
	if d.Time.IsZero() {
		return nil, nil
	}

	return []byte("\"" + time.Time(d.Time).Format(constanta.DefaultDtoOutTimeFormat) + "\""), nil
}
