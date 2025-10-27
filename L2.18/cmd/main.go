package main

import (
	"calendar/internal/config"
	"calendar/internal/di"
	"calendar/internal/logger"
	"calendar/internal/repository"
	"calendar/internal/web"

	"go.uber.org/fx"
)

func main() {
	app := fx.New(
		fx.Provide(
			config.MustLoad,
			logger.ProvideLogger,
			repository.NewInMemoryRepo,
			func(repo *repository.InMemoryRepo) repository.Storage {
				return repo
			},
			web.NewCalendarHandler,
		),

		fx.Invoke(
			di.StartHttpServer,
		),
	)
	app.Run()
}
