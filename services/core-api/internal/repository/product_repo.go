package repository

import (
	"context"
	"corestore/api/internal/domain"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ProductRepository struct {
	db *pgxpool.Pool
}

func NewProductRepository(db *pgxpool.Pool) *ProductRepository {
	return &ProductRepository{db: db}
}

// InitTable создает таблицу products, если её еще нет
func (r *ProductRepository) InitTable(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS products (
		id SERIAL PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		price NUMERIC(10, 2) NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`
	_, err := r.db.Exec(ctx, query)
	return err
}

// Create сохраняет новый товар в базу
func (r *ProductRepository) Create(ctx context.Context, name string, price float64) (*domain.Product, error) {
	query := `INSERT INTO products (name, price) VALUES ($1, $2) RETURNING id, name, price, created_at`

	p := &domain.Product{}
	err := r.db.QueryRow(ctx, query, name, price).Scan(&p.ID, &p.Name, &p.Price, &p.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create product: %w", err)
	}

	return p, nil
}

// GetAll возвращает список всех товаров
func (r *ProductRepository) GetAll(ctx context.Context) ([]domain.Product, error) {
	query := `SELECT id, name, price, created_at FROM products ORDER BY id DESC`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get products: %w", err)
	}
	defer rows.Close()

	var products []domain.Product
	for rows.Next() {
		var p domain.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan product: %w", err)
		}
		products = append(products, p)
	}

	return products, nil
}
