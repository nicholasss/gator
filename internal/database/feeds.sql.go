// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: feeds.sql

package database

import (
	"context"
	"time"

	"github.com/google/uuid"
)

const createFeed = `-- name: CreateFeed :one
insert into feeds (
	id, name, created_at, updated_at, url, user_id
) values (
	$1, $2, $3, $4, $5, $6
	)
returning id, name, created_at, updated_at, url, user_id
`

type CreateFeedParams struct {
	ID        uuid.UUID
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
	Url       string
	UserID    uuid.UUID
}

type CreateFeedRow struct {
	ID        uuid.UUID
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
	Url       string
	UserID    uuid.UUID
}

func (q *Queries) CreateFeed(ctx context.Context, arg CreateFeedParams) (CreateFeedRow, error) {
	row := q.db.QueryRowContext(ctx, createFeed,
		arg.ID,
		arg.Name,
		arg.CreatedAt,
		arg.UpdatedAt,
		arg.Url,
		arg.UserID,
	)
	var i CreateFeedRow
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.Url,
		&i.UserID,
	)
	return i, err
}

const getAllFeeds = `-- name: GetAllFeeds :many
select id, created_at, updated_at, name, url, user_id from feeds
`

func (q *Queries) GetAllFeeds(ctx context.Context) ([]Feed, error) {
	rows, err := q.db.QueryContext(ctx, getAllFeeds)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Feed
	for rows.Next() {
		var i Feed
		if err := rows.Scan(
			&i.ID,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.Name,
			&i.Url,
			&i.UserID,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getFeedName = `-- name: GetFeedName :one
select id, created_at, updated_at, name, url, user_id from feeds
	where name = $1
	limit 1
`

func (q *Queries) GetFeedName(ctx context.Context, name string) (Feed, error) {
	row := q.db.QueryRowContext(ctx, getFeedName, name)
	var i Feed
	err := row.Scan(
		&i.ID,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.Name,
		&i.Url,
		&i.UserID,
	)
	return i, err
}

const getFeedURL = `-- name: GetFeedURL :one
select id, created_at, updated_at, name, url, user_id from feeds
	where url = $1
	limit 1
`

func (q *Queries) GetFeedURL(ctx context.Context, url string) (Feed, error) {
	row := q.db.QueryRowContext(ctx, getFeedURL, url)
	var i Feed
	err := row.Scan(
		&i.ID,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.Name,
		&i.Url,
		&i.UserID,
	)
	return i, err
}

const getUsersFeeds = `-- name: GetUsersFeeds :many
select id, created_at, updated_at, name, url, user_id from feeds
	where user_id = $1
`

func (q *Queries) GetUsersFeeds(ctx context.Context, userID uuid.UUID) ([]Feed, error) {
	rows, err := q.db.QueryContext(ctx, getUsersFeeds, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Feed
	for rows.Next() {
		var i Feed
		if err := rows.Scan(
			&i.ID,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.Name,
			&i.Url,
			&i.UserID,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
