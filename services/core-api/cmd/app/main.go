package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5"
)

func main() {
	// 1. Читаем строку подключения к БД (или используем дефолтную для нашей песочницы из docker-compose)
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:secretpassword@localhost:5432/corestore?sslmode=disable"
	}

	// 2. Инициализируем контекст с таймаутом для подключения к БД
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Подключаемся к PostgreSQL через pgx
	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		log.Fatalf("❌ Не удалось подключиться к базе данных: %v", err)
	}
	defer conn.Close(context.Background())

	// Проверяем пинг базы данных
	if err := conn.Ping(ctx); err != nil {
		log.Fatalf("❌ База данных не отвечает: %v", err)
	}
	log.Println("✅ Успешное подключение к PostgreSQL!")

	// 3. Настраиваем роутер Chi
	r := chi.NewRouter()

	// Подключаем базовые мидлвары (логирование, паника-рекавери)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Эндпоинт для проверки здоровья сервиса (Health Check)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok", "service": "core-api"}`))
	})

	// 4. Запускаем HTTP-сервер с поддержкой graceful shutdown (мягкой остановки)
	port := ":8080"
	server := &http.Server{
		Addr:    port,
		Handler: r,
	}

	// Запуск в горутине, чтобы сервер не блокировал основной поток
	go func() {
		log.Printf("🚀 Сервер запущен на порту %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Ошибка запуска сервера: %v", err)
		}
	}()

	// Ожидание сигнала остановки (Ctrl+C)
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
