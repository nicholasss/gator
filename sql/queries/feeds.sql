-- name: CreateFeed :one
insert into feeds (
	id, name, created_at, updated_at, url, user_id
) values (
	$1, $2, $3, $4, $5, $6
	)
returning id, name, created_at, updated_at, url, user_id;

-- name: GetAllFeeds :many
select * from feeds;

-- name: GetFeedByName :one
select * from feeds
	where name = $1
	limit 1;

-- name: GetFeedByID :one
select * from feeds
	where id = $1
	limit 1;

-- name: GetFeedByURL :one
select * from feeds
	where url = $1
	limit 1;

-- name: GetFeedsByUser :many
select * from feeds
	where user_id = $1;

-- name: MarkFeedFetched :exec
update feeds
set last_fetched_at = $2,
	updated_at = $2
where id = $1;

-- name: GetNextFeedToFetch :many
select * from feeds
	order by last_fetched_at asc nulls first;
