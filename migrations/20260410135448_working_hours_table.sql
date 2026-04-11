-- +goose Up
CREATE TABLE working_hours(
	id serial PRIMARY KEY not null,
	start_date timestamp not null,
	end_date timestamp  not null,
	break_start timestamp,
	break_length interval
);

-- +goose Down
DROP TABLE working_hours;
