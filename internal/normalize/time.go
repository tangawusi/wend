package normalize

import "time"

func NormalizeTime(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	u := t.UTC().Truncate(time.Second)
	return &u
}
