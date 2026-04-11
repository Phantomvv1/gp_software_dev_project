-- +goose Up
CREATE TABLE visits (
	id serial primary key not null,
	start_time timestamp not null,
	visit_time interval not null,
	patient_id int references patients(id),
	doctor_id int references doctors(id),
);

-- +goose Down
DROP TABLE visits;
