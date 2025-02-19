-- name: CreatePost :one
insert into posts (
	id, created_at, updated_at, title, url, description, published_at, feed_id
) values (
	$1, $2, $3, $4, $5, $6, $7, $8
) returning *;

-- name: GetPostsForUser :many
select *
	from posts
	inner join feed_follows
	on feed_follows.feed_id = posts.feed_id
	where feed_follows.user_id = $1
	order by published_at asc
	limit $2;
