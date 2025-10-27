package repository

import (
	"calendar/internal/app"
	"fmt"
	"github.com/google/uuid"
	"sync"
	"time"
)

type Storage interface {
	Save(er *app.EventRequest) (*app.Event, error)
	Delete(er *app.EventRequest) error
	Update(*app.EventRequest) (*app.Event, error)
	LoadDay(UserID int, Date time.Time) ([]*app.Event, error)
	LoadWeek(UserID int, WeekStart time.Time) ([]*app.Event, error)
	LoadMonth(UserID int, MonthStart time.Time) ([]*app.Event, error)
}

type InMemoryRepo struct {
	Repo map[int][]*app.Event
	mu   sync.Mutex // Для обеспечения потокобезопасности
}

func NewInMemoryRepo() *InMemoryRepo {
	return &InMemoryRepo{
		Repo: make(map[int][]*app.Event),
	}
}

func (r *InMemoryRepo) Save(er *app.EventRequest) (*app.Event, error) {
	e, err := app.NewEvent(er)

	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	r.Repo[e.UserID] = append(r.Repo[e.UserID], e)
	r.mu.Unlock()
	return e, nil
}

func (r *InMemoryRepo) Delete(er *app.EventRequest) error {

	uid, err := uuid.Parse(er.EventId)
	if err != nil {
		return fmt.Errorf("%w: %v", app.ErrInvalidInput, err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	events := r.Repo[er.UserID]
	for i, event := range events {
		if event.EventId == uid {
			r.Repo[er.UserID] = append(events[:i], events[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("%w: %v", app.ErrBusinessLogic, "event not found")
}

func (r *InMemoryRepo) Update(e *app.EventRequest) (*app.Event, error) {
	if e.Date == "" && e.EventText == "" {
		return nil, fmt.Errorf("%w: %v", app.ErrBusinessLogic, "nothing to update")
	}

	uid, err := uuid.Parse(e.EventId)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", app.ErrInvalidInput, err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	events := r.Repo[e.UserID]
	for _, event := range events {
		if event.EventId == uid {
			err := event.Update(e.Date, e.EventText)
			if err != nil {
				return nil, err
			}
			return event, nil
		}
	}
	return nil, fmt.Errorf("%w: %v", app.ErrBusinessLogic, "event not found")
}

func (r *InMemoryRepo) LoadDay(UserID int, Date time.Time) ([]*app.Event, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	events := r.Repo[UserID]
	var result []*app.Event
	for _, event := range events {
		if event.Date.Equal(Date) {
			result = append(result, event)
		}
	}
	return result, nil
}

func (r *InMemoryRepo) LoadWeek(UserID int, Date time.Time) ([]*app.Event, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	events := r.Repo[UserID]
	Dy, Dw := Date.ISOWeek()
	var result []*app.Event
	for _, event := range events {
		Ey, Ew := event.Date.ISOWeek()
		if Ey == Dy && Ew == Dw {
			result = append(result, event)
		}
	}
	return result, nil
}

func (r *InMemoryRepo) LoadMonth(UserID int, Date time.Time) ([]*app.Event, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	events := r.Repo[UserID]
	var result []*app.Event
	for _, event := range events {
		if event.Date.Month() == Date.Month() && event.Date.Year() == Date.Year() {
			result = append(result, event)
		}
	}
	return result, nil
}
