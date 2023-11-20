// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.23.0
// source: query.sql

package db

import (
	"context"
)

const createNeed = `-- name: CreateNeed :exec
INSERT INTO needs (
    name
) VALUES ($1)
`

func (q *Queries) CreateNeed(ctx context.Context, name string) error {
	_, err := q.db.ExecContext(ctx, createNeed, name)
	return err
}

const listNeeds = `-- name: ListNeeds :many
SELECT name
FROM needs
ORDER BY name
`

func (q *Queries) ListNeeds(ctx context.Context) ([]string, error) {
	rows, err := q.db.QueryContext(ctx, listNeeds)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		items = append(items, name)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
