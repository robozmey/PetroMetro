package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/robozmey/podzemka/backend/internal/domain"
)

var ErrNotFound = errors.New("not found")

type PostgresRepository struct {
	db *sql.DB
}

func OpenPostgres(ctx context.Context) (*sql.DB, error) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return nil, errors.New("DATABASE_URL is not set")
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, wrapPostgresError("ping postgres", err)
	}

	return db, nil
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) ListCities(ctx context.Context) ([]domain.City, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, slug
		FROM cities
		ORDER BY id
	`)
	if err != nil {
		return nil, wrapPostgresError("list cities", err)
	}
	defer rows.Close()

	cities := make([]domain.City, 0)
	for rows.Next() {
		var city domain.City
		if err := rows.Scan(&city.ID, &city.Name, &city.Slug); err != nil {
			return nil, wrapPostgresError("scan city", err)
		}
		cities = append(cities, city)
	}
	if err := rows.Err(); err != nil {
		return nil, wrapPostgresError("iterate cities", err)
	}

	return cities, nil
}

func (r *PostgresRepository) CityExists(ctx context.Context, slug string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM cities WHERE slug = $1)`, slug).Scan(&exists)
	if err != nil {
		return false, wrapPostgresError("check city exists", err)
	}
	return exists, nil
}

func (r *PostgresRepository) GetCityBySlug(ctx context.Context, slug string) (domain.City, error) {
	var city domain.City
	err := r.db.QueryRowContext(ctx, `
		SELECT id, name, slug
		FROM cities
		WHERE slug = $1
	`, slug).Scan(&city.ID, &city.Name, &city.Slug)

	if errors.Is(err, sql.ErrNoRows) {
		return domain.City{}, ErrNotFound
	}
	if err != nil {
		return domain.City{}, wrapPostgresError("get city by slug", err)
	}

	return city, nil
}

func (r *PostgresRepository) GetSchemaBySlug(ctx context.Context, slug string) (domain.MetroSchema, error) {
	var schema domain.MetroSchema
	var data []byte
	err := r.db.QueryRowContext(ctx, `
		SELECT c.id, c.name, c.slug, m.schema
		FROM cities c
		JOIN metro_schemas m ON m.city_id = c.id
		WHERE c.slug = $1
	`, slug).Scan(&schema.City.ID, &schema.City.Name, &schema.City.Slug, &data)

	if errors.Is(err, sql.ErrNoRows) {
		return schema, ErrNotFound
	}
	if err != nil {
		return schema, wrapPostgresError("get schema", err)
	}

	var content domain.MetroContent
	if err := json.Unmarshal(data, &content); err != nil {
		return schema, fmt.Errorf("decode schema: %w", err)
	}
	if err := content.Validate(); err != nil {
		return schema, fmt.Errorf("invalid stored schema: %w", err)
	}

	schema.Stations, schema.Lines = content.Stations, content.Lines
	return schema, nil
}

func (r *PostgresRepository) SaveSchema(ctx context.Context, schema domain.MetroSchema) error {
	if schema.City.Name == "" || schema.City.Slug == "" {
		return errors.New("city name and slug are required")
	}

	content := domain.MetroContent{Stations: schema.Stations, Lines: schema.Lines}
	if err := content.Validate(); err != nil {
		return err
	}

	data, err := json.Marshal(content)
	if err != nil {
		return fmt.Errorf("encode schema: %w", err)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return wrapPostgresError("begin save schema", err)
	}
	defer tx.Rollback()

	var cityID int
	err = tx.QueryRowContext(ctx, `
		INSERT INTO cities (name, slug) 
		VALUES ($1, $2)
 		ON CONFLICT (slug) 
		DO UPDATE SET name = EXCLUDED.name 
		RETURNING id
		`, schema.City.Name, schema.City.Slug).Scan(&cityID)

	if err != nil {
		return wrapPostgresError("save city", err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO metro_schemas (city_id, schema) 
		VALUES ($1, $2::jsonb)
 		ON CONFLICT (city_id) 
		DO UPDATE SET schema = EXCLUDED.schema, 
					  updated_at = now()
		`, cityID, string(data))

	if err != nil {
		return wrapPostgresError("save schema", err)
	}
	return tx.Commit()
}

func wrapPostgresError(operation string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return fmt.Errorf("%s: postgres %s: %s", operation, pgErr.Code, pgErr.Message)
	}

	return fmt.Errorf("%s: %w", operation, err)
}
