package repository

import (
	"encoding/json"
	"testing"
	"time"
)

func TestDateUnmarshalJSON(t *testing.T) {
	var d Date
	if err := json.Unmarshal([]byte(`"2026-03-15"`), &d); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	want := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	if !d.Time.Equal(want) {
		t.Errorf("d.Time = %v, attendu %v", d.Time, want)
	}
}

func TestDateUnmarshalJSONNull(t *testing.T) {
	var d Date
	if err := json.Unmarshal([]byte(`null`), &d); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !d.Time.IsZero() {
		t.Errorf("d.Time = %v, attendu zéro", d.Time)
	}
}

func TestDateMarshalJSON(t *testing.T) {
	d := Date{Time: time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)}
	got, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if string(got) != `"2026-03-15"` {
		t.Errorf("Marshal = %s, attendu %q", got, `"2026-03-15"`)
	}
}

func TestDateToTimePtrAndBack(t *testing.T) {
	if dateToTimePtr(nil) != nil {
		t.Errorf("dateToTimePtr(nil) attendu nil")
	}
	if timePtrToDate(nil) != nil {
		t.Errorf("timePtrToDate(nil) attendu nil")
	}

	want := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	got := dateToTimePtr(timePtrToDate(&want))
	if got == nil || !got.Equal(want) {
		t.Errorf("aller-retour = %v, attendu %v", got, want)
	}
}
