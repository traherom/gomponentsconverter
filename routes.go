package gomponentsconverter

import (
	"context"
	_ "embed"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
)

func BuildRoutes(ctx context.Context) chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.GetHead)
	r.Use(httprate.LimitByIP(100, 1*time.Minute))
	r.Use(middleware.Recoverer)

	r.Handle("/", IndexHandler(ctx))

	r.Group(func(r chi.Router) {
		r.Use(middleware.NoCache)
		r.Handle("/convert", ConvertHandler())
	})

	return r
}
