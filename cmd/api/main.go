package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/alexedwards/scs/pgxstore"
	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/matheusburey/api-restful-go/internal/api"
	"github.com/matheusburey/api-restful-go/internal/services"
)

func main() {
	if err := godotenv.Load(); err != nil {
		panic(err)
	}

	ctx := context.Background()
	database_url := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_NAME"))

	p, err := pgxpool.New(ctx, database_url)
	if err != nil {
		panic(err)
	}
	defer p.Close()
	if err := p.Ping(ctx); err != nil {
		panic(err)
	}
	s := scs.New()
	s.Store = pgxstore.New(p)
	s.Lifetime = 24 * time.Hour
	s.Cookie.HttpOnly = true
	s.Cookie.SameSite = http.SameSiteLaxMode

	api := api.Api{
		Router:       chi.NewMux(),
		UsersService: services.NewUsersService(p),
		Sessions:     s,
	}
	api.BindRoutes()

	fmt.Println("Server running on port 3333 🚀")
	http.ListenAndServe("localhost:3333", api.Router)
}
