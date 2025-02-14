-- +goose up
create table feeds(
	id uuid primary key,
	name text,
	url text unique,
	user_id uuid references users (id) on delete cascade
);

-- +goose down
drop table feeds;
