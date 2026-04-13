-- +goose Up
CREATE TABLE doctor_working_overrides (
	id serial PRIMARY KEY,
	doctor_id int REFERENCES doctors(id) ON DELETE CASCADE,

	start_datetime timestamp NOT NULL,
	end_datetime timestamp NOT NULL,

	-- if NULL → doctor is NOT working
	start_time time,
	end_time time,

	break_start time,
	break_end time
);

-- +goose Down
drop table doctor_working_overrides;
