-- +goose Up
create table if not exists doctors(
	id serial PRIMARY KEY not null, 
	name text not null,
	email text unique not null,
	password text not null,
	address text not null
);

-- +goose Down
drop table doctors;
