package repository

import (
	"calendar/internal/app"
	"testing"
	"time"
)

func newReq(uid int, date, text string) *app.EventRequest {
	return &app.EventRequest{
		UserID:    uid,
		Date:      date,
		EventText: text,
	}
}

func TestInMemoryRepoSaveDeleteUpdateLoad(t *testing.T) {
	r := NewInMemoryRepo()

	ev, err := r.Save(newReq(10, "2025-05-05", "hello"))
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	if ev.UserID != 10 {
		t.Fatalf("unexpected userid")
	}

	reqUpdate := &app.EventRequest{
		EventId:   ev.EventId.String(),
		UserID:    ev.UserID,
		Date:      "",
		EventText: "newtext",
	}
	ue, err := r.Update(reqUpdate)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if ue.EventText != "newtext" {
		t.Fatalf("Update did not change text")
	}

	_, err = r.Update(&app.EventRequest{EventId: ev.EventId.String(), UserID: ev.UserID})
	if err == nil {
		t.Fatalf("expected error when nothing to update")
	}

	dt, _ := time.Parse("2006-01-02", "2025-05-05")
	list, err := r.LoadDay(ev.UserID, dt)
	if err != nil {
		t.Fatalf("LoadDay err: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 event for LoadDay, got %d", len(list))
	}

	listW, err := r.LoadWeek(ev.UserID, dt)
	if err != nil {
		t.Fatalf("LoadWeek err: %v", err)
	}
	if len(listW) != 1 {
		t.Fatalf("expected 1 event for LoadWeek, got %d", len(listW))
	}

	listM, err := r.LoadMonth(ev.UserID, dt)
	if err != nil {
		t.Fatalf("LoadMonth err: %v", err)
	}
	if len(listM) != 1 {
		t.Fatalf("expected 1 event for LoadMonth, got %d", len(listM))
	}

	if err := r.Delete(&app.EventRequest{EventId: ev.EventId.String(), UserID: ev.UserID}); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if err := r.Delete(&app.EventRequest{EventId: ev.EventId.String(), UserID: ev.UserID}); err == nil {
		t.Fatalf("expected error when deleting non-existent event")
	}
}

func TestInMemoryRepoUpdateInvalidUUID(t *testing.T) {
	r := NewInMemoryRepo()
	_, err := r.Update(&app.EventRequest{EventId: "bad-uuid", UserID: 1, Date: "2025-01-01"})
	if err == nil {
		t.Fatal("expected error for invalid uuid in Update")
	}
}

func TestInMemoryRepoDeleteInvalidUUID(t *testing.T) {
	r := NewInMemoryRepo()
	err := r.Delete(&app.EventRequest{EventId: "not-uuid", UserID: 1})
	if err == nil {
		t.Fatal("expected error for invalid uuid in Delete")
	}
}
