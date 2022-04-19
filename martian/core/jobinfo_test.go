package core

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/martian-lang/martian/martian/util"
)

func TestWallClockTime_UnmarshalJSON(t *testing.T) {
	var wt WallClockTime
	if err := json.Unmarshal([]byte(`"2022-04-19 12:01:03"`), &wt); err != nil {
		t.Error("legacy parse", err)
	}
	if tt, err := time.ParseInLocation(util.TIMEFMT, `2022-04-19 12:01:03`, time.Local); err != nil {
		t.Error("legacy parse string", err)
	} else if time.Time(wt).Sub(tt) != 0 {
		t.Error(wt, "!=", tt)
	}
	tt := WallClockTime(time.Now().Truncate(time.Millisecond))
	b, err := json.Marshal(tt)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &wt); err != nil {
		t.Error("iso parse", err)
	} else if time.Time(wt).Sub(time.Time(tt)) != 0 {
		t.Error(wt, "!=", tt)
	}
}
