package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

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

	api := api.Api{
		Router:       chi.NewMux(),
		UsersService: services.NewUsersService(p),
	}
	api.BindRoutes()

	fmt.Println("Server running on port 3333 🚀")
	http.ListenAndServe("localhost:3333", api.Router)
}
