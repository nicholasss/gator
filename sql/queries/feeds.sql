-- name: CreateFeed :one
insert into feeds (
	id, name, created_at, updated_at, url, user_id
) values (
	$1, $2, $3, $4, $5, $6
	)
returning id, name, created_at, updated_at, url, user_id;

-- name: GetAllFeeds :many
select * from feeds;

-- name: GetFeedName :one
select * from feeds
	where name = $1
	limit 1;

-- name: GetFeedURL :one
select * from feeds
	where url = $1
	limit 1;

-- name: GetUsersFeeds :many
select * from feeds
	where user_id = $1;
