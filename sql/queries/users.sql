-- name: CreateUser :one
insert into users (
	id, created_at, updated_at, name
) values (
	$1, $2, $3, $4
) returning id, created_at, updated_at, name;

-- name: GetUserByName :one
select * from users
	where name = $1
	limit 1;

-- name: GetUserByID :one
select * from users
	where id = $1
	limit 1;

-- name: GetUsers :many
select * from users;

-- name: ResetUsers :exec
delete from users;
