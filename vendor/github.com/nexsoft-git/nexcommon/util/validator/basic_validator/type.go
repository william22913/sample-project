package basic_validator

import "time"

type BasicValidator struct {
}

type Duration struct {
	Y  int
	M  int
	W  int
	D  int
	TH int
	TM int
	TS int
}

func (d Duration) timeDuration() time.Duration {
	var dur time.Duration

	dur = dur + (time.Duration(d.Y) * 365 * 24 * time.Hour)
	dur = dur + (time.Duration(d.M) * 30 * 24 * time.Hour)
	dur = dur + (time.Duration(d.W) * 7 * 24 * time.Hour)
	dur = dur + (time.Duration(d.D) * 24 * time.Hour)
	dur = dur + (time.Duration(d.TH) * time.Hour)
	dur = dur + (time.Duration(d.TM) * time.Minute)
	dur = dur + (time.Duration(d.TS) * time.Second)
	return dur
}
