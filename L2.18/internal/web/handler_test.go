package web

import (
	"bytes"
	"calendar/internal/app"
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// mockRepo allows controlling behavior per-method for tests
type mockRepo struct {
	SaveFn     func(er *app.EventRequest) (*app.Event, error)
	UpdateFn   func(er *app.EventRequest) (*app.Event, error)
	DeleteFn   func(er *app.EventRequest) error
	LoadDayFn  func(UserID int, Date time.Time) ([]*app.Event, error)
	LoadWeekFn func(UserID int, Date time.Time) ([]*app.Event, error)
	LoadMonFn  func(UserID int, Date time.Time) ([]*app.Event, error)
}

func (m *mockRepo) Save(er *app.EventRequest) (*app.Event, error) {
	return m.SaveFn(er)
}
func (m *mockRepo) Delete(er *app.EventRequest) error {
	return m.DeleteFn(er)
}
func (m *mockRepo) Update(er *app.EventRequest) (*app.Event, error) {
	return m.UpdateFn(er)
}
func (m *mockRepo) LoadDay(UserID int, Date time.Time) ([]*app.Event, error) {
	return m.LoadDayFn(UserID, Date)
}
func (m *mockRepo) LoadWeek(UserID int, Date time.Time) ([]*app.Event, error) {
	return m.LoadWeekFn(UserID, Date)
}
func (m *mockRepo) LoadMonth(UserID int, Date time.Time) ([]*app.Event, error) {
	return m.LoadMonFn(UserID, Date)
}

func TestCreateEventOK(t *testing.T) {
	logger := zap.NewNop()
	ev := &app.Event{
		UserID:    1,
		EventText: "x",
	}
	mock := &mockRepo{
		SaveFn: func(er *app.EventRequest) (*app.Event, error) {
			return ev, nil
		},
	}
	h := NewCalendarHandler(mock, logger)

	body, _ := json.Marshal(app.EventRequest{UserID: 1, Date: "2025-01-01", EventText: "x"})
	req := httptest.NewRequest(http.MethodPost, "/create", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.CreateEvent(w, req)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 got %d", resp.StatusCode)
	}
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, ok := out["result"]; !ok {
		t.Fatalf("missing result in response")
	}
}

func TestCreateEventBadJSON(t *testing.T) {
	logger := zap.NewNop()
	mock := &mockRepo{}
	h := NewCalendarHandler(mock, logger)
	req := httptest.NewRequest(http.MethodPost, "/create", bytes.NewReader([]byte("{")))
	w := httptest.NewRecorder()
	h.CreateEvent(w, req)
	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad json, got %d", resp.StatusCode)
	}
}

func TestDeleteEventBehavior(t *testing.T) {
	logger := zap.NewNop()
	// success
	mock := &mockRepo{
		DeleteFn: func(er *app.EventRequest) error {
			return nil
		},
	}
	h := NewCalendarHandler(mock, logger)
	body, _ := json.Marshal(app.EventRequest{EventId: "a", UserID: 1})
	req := httptest.NewRequest(http.MethodPost, "/delete", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.DeleteEvent(w, req)
	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for delete success, got %d", w.Result().StatusCode)
	}

	// -> 503
	mock2 := &mockRepo{
		DeleteFn: func(er *app.EventRequest) error {
			return app.ErrBusinessLogic
		},
	}
	h2 := NewCalendarHandler(mock2, logger)
	body2, _ := json.Marshal(app.EventRequest{EventId: "a", UserID: 1})
	req2 := httptest.NewRequest(http.MethodPost, "/delete", bytes.NewReader(body2))
	w2 := httptest.NewRecorder()
	h2.DeleteEvent(w2, req2)
	if w2.Result().StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when repo returns ErrBusinessLogic")
	}

	// -> 400

	mock3 := &mockRepo{
		DeleteFn: func(er *app.EventRequest) error {
			return nil
		},
	}
	h3 := NewCalendarHandler(mock3, logger)
	req3 := httptest.NewRequest(http.MethodPost, "/delete", bytes.NewReader(nil))
	w3 := httptest.NewRecorder()
	h3.DeleteEvent(w3, req3)
	if w3.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for delete success, got %d", w3.Result().StatusCode)
	}
}

func TestUpdateEventSuccess(t *testing.T) {
	logger := zap.NewNop()
	expected := &app.Event{
		EventId:   uuid.New(),
		UserID:    1,
		EventText: "updated",
	}
	mock := &mockRepo{
		UpdateFn: func(er *app.EventRequest) (*app.Event, error) {
			return expected, nil
		},
	}
	h := NewCalendarHandler(mock, logger)

	body, _ := json.Marshal(app.EventRequest{
		EventId:   "some-uuid",
		UserID:    1,
		Date:      "2025-01-01",
		EventText: "updated",
	})
	req := httptest.NewRequest(http.MethodPut, "/update", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.UpdateEvent(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for success, got %d", w.Result().StatusCode)
	}
}

func TestUpdateEventErrors(t *testing.T) {
	logger := zap.NewNop()
	tests := []struct {
		name       string
		body       string
		mockError  error
		wantStatus int
	}{
		{
			name:       "invalid json",
			body:       "{invalid",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "business logic error",
			body:       `{"event_id":"some-id","user_id":1}`,
			mockError:  app.ErrBusinessLogic,
			wantStatus: http.StatusServiceUnavailable,
		},
		{
			name:       "invalid input error",
			body:       `{"event_id":"some-id","user_id":1}`,
			mockError:  app.ErrInvalidInput,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockRepo{
				UpdateFn: func(er *app.EventRequest) (*app.Event, error) {
					return nil, tt.mockError
				},
			}
			h := NewCalendarHandler(mock, logger)

			req := httptest.NewRequest(http.MethodPut, "/update", strings.NewReader(tt.body))
			w := httptest.NewRecorder()
			h.UpdateEvent(w, req)

			if w.Result().StatusCode != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Result().StatusCode)
			}
		})
	}
}

func TestEventsInvalidUserOrDate(t *testing.T) {
	logger := zap.NewNop()
	mock := &mockRepo{}
	h := NewCalendarHandler(mock, logger)

	// invalid user
	req := httptest.NewRequest(http.MethodGet, "/day?user_id=bad&date=2025-01-01", nil)
	w := httptest.NewRecorder()
	f := h.eventsHandler(h.repo.LoadDay, "Day")
	f.ServeHTTP(w, req)
	if w.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad user_id")
	}

	// invalid date
	req = httptest.NewRequest(http.MethodGet, "/day?user_id=1&date=bad", nil)
	w = httptest.NewRecorder()
	f.ServeHTTP(w, req)
	if w.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad date")
	}
}

func TestEventsSuccessAndRepoErrors(t *testing.T) {
	logger := zap.NewNop()
	// success
	mock := &mockRepo{
		LoadDayFn: func(UserID int, Date time.Time) ([]*app.Event, error) {
			return []*app.Event{{UserID: UserID, EventText: "ok"}}, nil
		},
	}
	h := NewCalendarHandler(mock, logger)
	req := httptest.NewRequest(http.MethodGet, "/day?user_id=1&date=2025-01-01", nil)
	w := httptest.NewRecorder()
	f := h.eventsHandler(h.repo.LoadDay, "Day")
	f.ServeHTTP(w, req)
	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for success")
	}
	// -> 400
	mock2 := &mockRepo{
		LoadDayFn: func(UserID int, Date time.Time) ([]*app.Event, error) {
			return nil, app.ErrInvalidInput
		},
	}
	h2 := NewCalendarHandler(mock2, logger)
	req2 := httptest.NewRequest(http.MethodGet, "/day?user_id=a&date=2025-01-01", nil)
	w2 := httptest.NewRecorder()
	f = h2.eventsHandler(h.repo.LoadDay, "Day")
	f.ServeHTTP(w2, req2)
	if w2.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 when repo returns ErrInvalidInput")
	}
	// -> 503
	mock3 := &mockRepo{
		LoadDayFn: func(UserID int, Date time.Time) ([]*app.Event, error) {
			return nil, app.ErrBusinessLogic
		},
	}
	h3 := NewCalendarHandler(mock3, logger)
	req3 := httptest.NewRequest(http.MethodGet, "/day?user_id=1&date=2025-01-01", nil)
	w3 := httptest.NewRecorder()
	f = h.eventsHandler(h3.repo.LoadDay, "Day")
	f.ServeHTTP(w3, req3)
	if w3.Result().StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when repo returns business error")
	}
}

func TestLoggerMiddleware(t *testing.T) {
	core, logs := observer.New(zap.InfoLevel)
	logger := zap.New(core)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	r := chi.NewRouter()
	r.Use(LoggerMiddleware(logger))
	r.Get("/test", testHandler)

	req := httptest.NewRequest("GET", "/test?param=value", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	entries := logs.All()
	if len(entries) != 1 {
		t.Fatalf("Expected 1 log entry, got %d", len(entries))
	}

	entry := entries[0]
	fields := entry.ContextMap()

	tests := []struct {
		field    string
		expected string
	}{
		{"method", "GET"},
		{"url", "/test?param=value"},
	}

	for _, tt := range tests {
		if got := fields[tt.field]; got != tt.expected {
			t.Errorf("Expected %s to be %v, got %v", tt.field, tt.expected, got)
		}
	}

	duration, ok := fields["duration"].(time.Duration)
	if !ok {
		t.Error("Duration field not found or wrong type")
	} else if duration < 10*time.Millisecond {
		t.Errorf("Duration too short: %v", duration)
	}
}
