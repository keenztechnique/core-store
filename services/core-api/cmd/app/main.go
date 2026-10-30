package main

import (
	"context"
	"corestore/api/internal/handler"
	"corestore/api/internal/repository"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9" // <-- Импортируем Redis
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:secretpassword@localhost:5432/corestore?sslmode=disable"
	}

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1. Подключение к PostgreSQL
	dbPool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatalf("❌ Не удалось создать пул соединений с БД: %v", err)
	}
	defer dbPool.Close()

	if err := dbPool.Ping(ctx); err != nil {
		log.Fatalf("❌ База данных не отвечает: %v", err)
	}
	log.Println("✅ Успешное подключение к PostgreSQL!")

	// 2. Подключение к Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "", // нет пароля по умолчанию в нашей докер-инфраструктуре
		DB:       0,  // дефолтная база
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("❌ Redis не отвечает: %v", err)
	}
	log.Println("✅ Успешное подключение к Redis!")

	// Инициализация репозиториев и хендлеров
	productRepo := repository.NewProductRepository(dbPool)

	if err := productRepo.InitTable(ctx); err != nil {
		log.Fatalf("❌ Не удалось инициализировать таблицу products: %v", err)
	}

	// Передаем и базу, и Redis в хендлер
	productHandler := handler.NewProductHandler(productRepo, rdb)

	// Роутер Chi
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok", "service": "core-api"}`))
	})

	r.Route("/products", func(r chi.Router) {
		r.Post("/", productHandler.CreateProduct)
		r.Get("/", productHandler.GetProducts)
	})

	port := ":8080"
	server := &http.Server{
		Addr:    port,
		Handler: r,
	}

	go func() {
		log.Printf("🚀 Сервер запущен на порту %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Ошибка запуска сервера: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Println("🛑 Остановка сервера...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("❌ Принудительная остановка сервера: %v", err)
	}

	log.Println("👋 Сервер успешно остановлен.")
}
