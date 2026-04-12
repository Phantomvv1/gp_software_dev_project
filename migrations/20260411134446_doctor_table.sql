-- +goose Up
create table if not exists doctors(
	id serial PRIMARY KEY not null, 
	name text not null,
	email text unique not null,
	password text not null,
	address text not null,
	working_hours_id int REFERENCES working_hours(id) on delete cascade
);

-- +goose Down
drop table doctors;
