-- +goose up
create table feeds (
	id uuid primary key,
	created_at timestamp not null,
	updated_at timestamp not null,
	name text not null,
	url text unique not null,
	user_id uuid not null
);

alter table feeds
	add constraint fk_user
	foreign key (user_id)
	references users(id)
	on delete cascade;

-- +goose down
drop table feeds;
