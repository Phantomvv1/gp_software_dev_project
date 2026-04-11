-- +goose Up
create table if not exists doctors(
	id serial PRIMARY KEY not null, 
	name text not null,
	email text unique not null,
	address text not null,
	working_hours_id int FOREIGN KEY REFERENCES working_hours(id)
);

-- +goose Down
drop table doctors;
