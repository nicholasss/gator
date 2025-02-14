-- name: CreateFeed :one
insert into feeds (
	id, name, created_at, updated_at, url, user_id
) values (
	$1, $2, $3, $4, $5, $6
	)
returning id, name, created_at, updated_at, url, user_id;
