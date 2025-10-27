package web

import (
	_ "calendar/docs"
	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger"
)

func RegisterRoutes(r chi.Router, h *CalendarHandler) {
	r.Group(func(r chi.Router) {
		r.Use(LoggerMiddleware(h.logger))
		r.Post("/create_event", h.CreateEvent)
		r.Post("/update_event", h.UpdateEvent)
		r.Post("/delete_event", h.DeleteEvent)
		r.Get("/events_for_day", h.EventsForDay)
		r.Get("/events_for_week", h.EventsForWeek)
		r.Get("/events_for_month", h.EventsForMonth)
	})
	r.Get("/swagger/*", httpSwagger.WrapHandler)
}
