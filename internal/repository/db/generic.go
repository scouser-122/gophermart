package db

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// QueryExecutor interface for DB queries
type QueryExecutor interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// GenericRepository generic Postgres repository to work with any entity
type GenericRepository[T any] struct {
	db      QueryExecutor
	table   string
	keyName string
	mapper  func(row pgx.Row) (*T, error)
}

// NewGenericRepository creates generic repository
func NewGenericRepository[T any](db QueryExecutor, table string, keyName string, mapper func(row pgx.Row) (*T, error)) *GenericRepository[T] {
	return &GenericRepository[T]{
		db:      db,
		table:   table,
		keyName: keyName,
		mapper:  mapper,
	}
}

// Create creates entity, fields should be string of entity field names separated by comma
func (r *GenericRepository[T]) Create(ctx context.Context, fields string, args ...interface{}) (*T, error) {
	fieldNames := strings.Split(strings.ReplaceAll(fields, " ", ""), ",")
	var valuesArgs []string
	for i := 1; i <= len(fieldNames); i++ {
		valuesArgs = append(valuesArgs, fmt.Sprintf("$%d", i))
	}
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", r.table, fields, strings.Join(valuesArgs, ","))
	_, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	keyIndex := slices.Index(fieldNames, r.keyName)
	keyValue := args[keyIndex].(string)
	return r.GetByID(ctx, keyValue)
}

// Update updates entity
func (r *GenericRepository[T]) Update(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
	return r.db.Exec(ctx, query, args...)
}

// GetByID find entity by ID
func (r *GenericRepository[T]) GetByID(ctx context.Context, id string) (*T, error) {
	query := fmt.Sprintf("SELECT * FROM %s WHERE %s = $1", r.table, r.keyName)
	row := r.db.QueryRow(ctx, query, id)
	return r.mapper(row)
}

// GetAll returns all entities
func (r *GenericRepository[T]) GetAll(ctx context.Context, limit, offset int) ([]*T, error) {
	query := fmt.Sprintf("SELECT * FROM %s ORDER BY %s LIMIT $1 OFFSET $2", r.table, r.keyName)
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*T
	for rows.Next() {
		row := &pgxRowAdapter{rows: rows}
		entity, err := r.mapper(row)
		if err != nil {
			return nil, err
		}
		results = append(results, entity)
	}
	return results, rows.Err()
}

// GetAllConditional returns all entities which met condition ordered by specified field
func (r *GenericRepository[T]) GetAllConditional(
	ctx context.Context,
	condition string,
	conditionFields []any,
	orderBy string,
	limit, offset int,
) ([]*T, error) {
	var args []interface{}
	argCounter := 0
	for i := 0; i < len(conditionFields); i++ {
		argCounter++
		args = append(args, conditionFields[i])
	}
	args = append(args, limit)
	argCounter++
	limitArg := fmt.Sprintf("$%d", argCounter)

	args = append(args, offset)
	argCounter++
	offsetArg := fmt.Sprintf("$%d", argCounter)

	query := fmt.Sprintf("SELECT * FROM %s %s ORDER BY %s LIMIT %s OFFSET %s", r.table, condition, orderBy, limitArg, offsetArg)
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*T
	for rows.Next() {
		row := &pgxRowAdapter{rows: rows}
		entity, err := r.mapper(row)
		if err != nil {
			return nil, err
		}
		results = append(results, entity)
	}
	return results, rows.Err()
}

// CustomQuery makes custom query request and returns result
func (r *GenericRepository[T]) CustomQuery(
	ctx context.Context,
	mapper func(row pgx.Row) error,
	query string,
	args ...interface{},
) error {
	row := r.db.QueryRow(ctx, query, args...)
	return mapper(row)
}

// WithTx returns new generic repository using transaction
func (r *GenericRepository[T]) WithTx(tx pgx.Tx) *GenericRepository[T] {
	return NewGenericRepository(tx, r.table, r.keyName, r.mapper)
}

// pgxRowAdapter - adapter for pgx.Rows
type pgxRowAdapter struct {
	rows pgx.Rows
}

func (r *pgxRowAdapter) Scan(dest ...interface{}) error {
	return r.rows.Scan(dest...)
}
