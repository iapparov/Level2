package web

import (
	"calendar/internal/app"
	"calendar/internal/repository"
	"encoding/json"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"net/http"
	"strconv"
	"time"
)

type CalendarHandler struct {
	repo   repository.Storage
	logger *zap.Logger
}

func NewCalendarHandler(repo repository.Storage, logger *zap.Logger) *CalendarHandler {
	return &CalendarHandler{
		repo:   repo,
		logger: logger,
	}
}

// CreateEvent godoc
// @Summary Create event
// @Description Create new calendar event
// @Tags events
// @Accept json
// @Produce json
// @Param event body app.EventRequest true "Event to create"
// @Success 	 200 {object} app.Event "created event" // note: response wrapped as {"result": <app.Event>}
// @Failure 	 400  {object} ErrorResponse "invalid user_id or date"
// @Failure	 	 503  {object} ErrorResponse "service unavailable"
// @Failure 	 500  {object} ErrorResponse "internal server error"
// @Router /create_event [post]
func (h *CalendarHandler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	var er app.EventRequest
	if err := json.NewDecoder(r.Body).Decode(&er); err != nil {
		h.logger.Warn("invalid request body", zap.Error(err))
		writeError(w, "bad calendar request", 400)
		return
	}
	e, err := h.repo.Save(&er)
	if err != nil {
		errParser(w, h.logger, err, "save failed")
		return
	}
	h.logger.Info("event created", zap.String("event_id", e.EventId.String()))
	writeJson(w, e)
}

// UpdateEvent godoc
// @Summary Update event
// @Description Update existing calendar event (by event_id)
// @Tags events
// @Accept json
// @Produce json
// @Param event body app.EventRequest true "Event update request"
// @Success 	 200 {object} app.Event "updated event" // note: response wrapped as {"result": <app.Event>}
// @Failure 	 400  {object} ErrorResponse "invalid user_id or date"
// @Failure	 	 503  {object} ErrorResponse "service unavailable"
// @Failure 	 500  {object} ErrorResponse "internal server error"
// @Router /update_event [post]
func (h *CalendarHandler) UpdateEvent(w http.ResponseWriter, r *http.Request) {
	var er app.EventRequest
	if err := json.NewDecoder(r.Body).Decode(&er); err != nil {
		h.logger.Warn("invalid request body", zap.Error(err))
		writeError(w, "bad calendar update request", 400)
		return
	}
	e, err := h.repo.Update(&er)
	if err != nil {
		errParser(w, h.logger, err, "update failed")
		return
	}
	h.logger.Info("event updated", zap.String("event_id", e.EventId.String()))
	writeJson(w, e)
}

// DeleteEvent godoc
// @Summary Delete event
// @Description Delete event by event_id for given user
// @Tags events
// @Accept json
// @Produce json
// @Param event body app.EventRequest true "Event delete request (needs event_id and user_id)"
// @Success      200  {array}  app.EventRequest
// @Failure 	 400  {object} ErrorResponse "invalid user_id or date"
// @Failure	 	 503  {object} ErrorResponse "service unavailable"
// @Failure 	 500  {object} ErrorResponse "internal server error"
// @Router /delete_event [post]
func (h *CalendarHandler) DeleteEvent(w http.ResponseWriter, r *http.Request) {
	var er app.EventRequest
	if err := json.NewDecoder(r.Body).Decode(&er); err != nil {
		h.logger.Warn("invalid request body", zap.Error(err))
		writeError(w, "bad calendar delete request", http.StatusBadRequest)
		return
	}
	err := h.repo.Delete(&er)
	if err != nil {
		errParser(w, h.logger, err, "delete failed")
		return
	}
	h.logger.Info("event updated", zap.String("event_id", er.EventId))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	writeJson(w, er)
}

// EventsForDay godoc
// @Summary      Events for day
// @Description  Get events for a specific day for a user
// @Tags         events
// @Accept       json
// @Produce      json
// @Param        user_id  query  int     true  "User ID"
// @Param        date     query  string  true  "Date in format YYYY-MM-DD"
// @Success      200  {array}   app.Event  "list of events"
// @Failure 	 400  {object} ErrorResponse "invalid user_id or date"
// @Failure	 	 503  {object} ErrorResponse "service unavailable"
// @Failure 	 500  {object} ErrorResponse "internal server error"
// @Router       /events_for_day [get]
func (h *CalendarHandler) EventsForDay(w http.ResponseWriter, r *http.Request) {
	h.eventsHandler(h.repo.LoadDay, "Day")(w, r)
}

// EventsForWeek godoc
// @Summary      Events for week
// @Description  Get events for the ISO week that contains the given date
// @Tags         events
// @Accept       json
// @Produce      json
// @Param        user_id  query  int     true  "User ID"
// @Param        date     query  string  true  "Date in format YYYY-MM-DD (any day of the week)"
// @Success      200  {array}  app.Event
// @Failure 	 400  {object} ErrorResponse "invalid user_id or date"
// @Failure	 	 503  {object} ErrorResponse "service unavailable"
// @Failure 	 500  {object} ErrorResponse "internal server error"
// @Router       /events_for_week [get]
func (h *CalendarHandler) EventsForWeek(w http.ResponseWriter, r *http.Request) {
	h.eventsHandler(h.repo.LoadWeek, "Week")(w, r)
}

// EventsForMonth godoc
// @Summary      Events for month
// @Description  Get events for the month that contains the given date
// @Tags         events
// @Accept       json
// @Produce      json
// @Param        user_id  query  int     true  "User ID"
// @Param        date     query  string  true  "Date in format YYYY-MM-DD (any day of the month)"
// @Success      200  {array}   app.Event
// @Failure 	 400  {object} ErrorResponse "invalid user_id or date"
// @Failure	 	 503  {object} ErrorResponse "service unavailable"
// @Failure 	 500  {object} ErrorResponse "internal server error"
// @Router       /events_for_month [get]
func (h *CalendarHandler) EventsForMonth(w http.ResponseWriter, r *http.Request) {
	h.eventsHandler(h.repo.LoadMonth, "Month")(w, r)
}

func (h *CalendarHandler) eventsHandler(loadFunc func(user int, date time.Time) ([]*app.Event, error), period string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rq := r.URL.Query()

		user, err := strconv.Atoi(rq.Get("user_id"))
		if err != nil {
			h.logger.Warn("invalid user id", zap.Error(err))
			writeError(w, "invalid user_id", http.StatusBadRequest)
			return
		}

		date := rq.Get("date")
		d, err := app.TimeParser(date)
		if err != nil {
			h.logger.Warn("invalid date", zap.Error(err))
			writeError(w, "invalid date", http.StatusBadRequest)
			return
		}

		events, err := loadFunc(user, d)
		if err != nil {
			errParser(w, h.logger, err, fmt.Sprintf("events for %s load failed ", period))
			return
		}

		h.logger.Info("events fetched", zap.String("Period", period), zap.Int("user_id", user))
		writeJson(w, events)
	}
}

func errParser(w http.ResponseWriter, logger *zap.Logger, err error, msg string) {
	logger.Debug("get events for month failed", zap.Error(err))
	if errors.Is(err, app.ErrInvalidInput) {
		writeError(w, msg, http.StatusBadRequest)
	} else if errors.Is(err, app.ErrBusinessLogic) {
		writeError(w, msg, http.StatusServiceUnavailable)
	} else {
		writeError(w, msg, http.StatusInternalServerError)
	}
}

func writeJson(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]any{"result": payload}); err != nil {
		writeError(w, "failed to encode response", http.StatusInternalServerError)
	}
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func writeError(w http.ResponseWriter, msg string, code int) {
	var er ErrorResponse
	er.Error = msg
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(er.Error); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
