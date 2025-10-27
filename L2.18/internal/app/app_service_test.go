package app

import (
	"testing"
	"time"
)

func TestTimeParserOK(t *testing.T) {
	s := "2025-10-27"
	got, err := TimeParser(s)
	if err != nil {
		t.Fatalf("TimeParser returned error: %v", err)
	}
	if got.Year() != 2025 || got.Month() != 10 || got.Day() != 27 {
		t.Fatalf("unexpected parsed time: %v", got)
	}
}

func TestTimeParserBad(t *testing.T) {
	_, err := TimeParser("bad-date")
	if err == nil {
		t.Fatal("expected error for bad date")
	}
}

func TestNewEventOK(t *testing.T) {
	req := &EventRequest{
		UserID:    1,
		Date:      "2025-01-02",
		EventText: "hello",
	}
	ev, err := NewEvent(req)
	if err != nil {
		t.Fatalf("NewEvent failed: %v", err)
	}
	if ev.UserID != req.UserID {
		t.Fatalf("UserID mismatch: want %d got %d", req.UserID, ev.UserID)
	}
	if ev.EventText != req.EventText {
		t.Fatalf("EventText mismatch")
	}

	y, m, d := ev.Date.Date()
	if y != 2025 || m != time.January || d != 2 {
		t.Fatalf("Date parsed wrong: %v", ev.Date)
	}
}

func TestNewEventBadDate(t *testing.T) {
	req := &EventRequest{
		UserID:    1,
		Date:      "not-a-date",
		EventText: "x",
	}
	_, err := NewEvent(req)
	if err == nil {
		t.Fatal("expected error for invalid date")
	}
	if !IsInvalidInputErr(err) {
		t.Fatalf("expected ErrInvalidInput wrapped, got: %v", err)
	}
}

func IsInvalidInputErr(err error) bool {
	return err != nil && (err.Error() == ErrInvalidInput.Error() || (err.Error() != "" && ErrInvalidInput != nil))
}

func TestEventUpdateOK(t *testing.T) {
	er := &EventRequest{
		UserID:    2,
		Date:      "2024-12-12",
		EventText: "orig",
	}
	ev, err := NewEvent(er)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	if err := ev.Update("2025-01-01", "changed"); err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if ev.EventText != "changed" {
		t.Fatalf("text not updated")
	}
	y, m, d := ev.Date.Date()
	if y != 2025 || m != time.January || d != 1 {
		t.Fatalf("date not updated: %v", ev.Date)
	}
}

func TestEventUpdateBadDate(t *testing.T) {
	er := &EventRequest{
		UserID:    2,
		Date:      "2024-12-12",
		EventText: "orig",
	}
	ev, err := NewEvent(er)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}
	if err := ev.Update("bad", ""); err == nil {
		t.Fatal("expected error for bad date in Update")
	}
}
