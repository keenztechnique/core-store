package handler

import (
	"context"
	"corestore/api/internal/domain"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/redis/go-redis/v9"
)

type ProductRepo interface {
	Create(ctx context.Context, name string, price float64) (*domain.Product, error)
	GetAll(ctx context.Context) ([]domain.Product, error)
}

// Интерфейс для Kafka продюсера, чтобы код оставался тестируемым
type EventProducer interface {
	PublishProductCreated(ctx context.Context, product interface{}) error
}

type ProductHandler struct {
	repo     ProductRepo
	rdb      *redis.Client
	kafka    EventProducer
	validate *validator.Validate
}

func NewProductHandler(repo ProductRepo, rdb *redis.Client, kafka EventProducer) *ProductHandler {
	return &ProductHandler{
		repo:     repo,
		rdb:      rdb,
		kafka:    kafka,
		validate: validator.New(),
	}
}

type CreateProductRequest struct {
	Name  string  `json:"name" validate:"required,min=2,max=100"`
	Price float64 `json:"price" validate:"required,gt=0"`
}

func (h *ProductHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var req CreateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "validation failed: " + err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	product, err := h.repo.Create(ctx, req.Name, req.Price)
	if err != nil {
		http.Error(w, `{"error": "internal server error"}`, http.StatusInternalServerError)
		return
	}

	// 🗑️ Инвалидируем кэш в Redis
	_ = h.rdb.Del(ctx, "products:all")

	// 📨 Публикуем событие в Kafka (в реальных проектах это можно делать горутиной или через Outbox паттерн)
	go func() {
		bgCtx, bgCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer bgCancel()
		_ = h.kafka.PublishProductCreated(bgCtx, product)
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(product)
}

func (h *ProductHandler) GetProducts(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	cacheKey := "products:all"

	cachedData, err := h.rdb.Get(ctx, cacheKey).Result()
	if err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(cachedData))
		return
	}

	products, err := h.repo.GetAll(ctx)
	if err != nil {
		http.Error(w, `{"error": "internal server error"}`, http.StatusInternalServerError)
		return
	}

	respData, err := json.Marshal(products)
	if err == nil {
		_ = h.rdb.Set(ctx, cacheKey, respData, 60*time.Second)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	w.WriteHeader(http.StatusOK)
	w.Write(respData)
}
