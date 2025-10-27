package app

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"time"
)

type Event struct {
	EventId   uuid.UUID `json:"event_id"`
	UserID    int       `json:"user_id"`
	Date      time.Time `json:"date"`
	EventText string    `json:"event"`
}

type EventRequest struct {
	EventId   string `json:"event_id"`
	UserID    int    `json:"user_id"`
	Date      string `json:"date"`
	EventText string `json:"event"`
}

/*
// Можно сделать UserID типом uid.UUID, но исходя из т/з такой необходимости нет
*/

var (
	ErrInvalidInput  = errors.New("invalid input")
	ErrBusinessLogic = errors.New("business logic error")
)

func NewEvent(er *EventRequest) (*Event, error) {
	t, err := TimeParser(er.Date)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}
	return &Event{
		EventId:   uuid.New(),
		UserID:    er.UserID,
		Date:      t,
		EventText: er.EventText,
	}, nil
}

func (e *Event) Update(Date string, EventText string) error {
	if Date != "" {
		t, err := time.Parse("2006-01-02", Date)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidInput, err)
		}
		e.Date = t
	}
	if EventText != "" {
		e.EventText = EventText
	}
	return nil
}

func TimeParser(date string) (time.Time, error) { // в Repo тоже парсится время, если мы решим изменить формат, то поменяем только в этой функции
	t, err := time.Parse("2006-01-02", date)
	return t, err
}
