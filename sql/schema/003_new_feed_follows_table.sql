-- +goose Up
create table feed_follows (
	id uuid primary key,
	created_at timestamp not null,
	updated_at timestamp not null,
	user_id uuid not null,
	feed_id uuid not null,

	unique(user_id, feed_id)
);

alter table feed_follows
	add constraint fk_user
	foreign key (user_id)
	references users(id)
	on delete cascade;

alter table feed_follows
	add constraint fk_feed
	foreign key (feed_id)
	references feeds(id)
	on delete cascade;

-- +goose Down
drop table feed_follows;
