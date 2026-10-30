package handler

import (
	"context"
	"corestore/api/internal/domain"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10" // <-- Импортируем валидатор
)

type ProductRepo interface {
	Create(ctx context.Context, name string, price float64) (*domain.Product, error)
	GetAll(ctx context.Context) ([]domain.Product, error)
}

type ProductHandler struct {
	repo ProductRepo
	validate *validator.Validate // <-- Добавляем поле для валидатора
}

func NewProductHandler(repo ProductRepo) *ProductHandler {
	return &ProductHandler{
		repo:     repo,
		validate: validator.New(), // <-- Инициализируем при создании хендлера
	}
}

type CreateProductRequest struct {
	Name  string  `json:"name" validate:"required,min=2,max=100"`
	Price float64 `json:"price" validate:"required,gt=0"`
}

func (h *ProductHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var req CreateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" || req.Price <= 0 {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	// 🔍 Выполняем валидацию по нашим тегам
	if err := h.validate.Struct(req); err != nil {
		// Возвращаем понятную ошибку клиенту, если валидация не прошла
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(product)
}

func (h *ProductHandler) GetProducts(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	products, err := h.repo.GetAll(ctx)
	if err != nil {
		http.Error(w, `{"error": "internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(products)
}
