-- +goose Up
CREATE TABLE patients (
	id serial primary key not null,
	name text not null,
	email text unique not null,
	password text not null,
	phone_number text unique not null,
	doctor_id int REFERENCES doctors(id) on delete cascade
);

-- +goose Down
DROP TABLE patients;
