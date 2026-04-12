-- +goose Up
CREATE TABLE normal_working_hours(
	id serial PRIMARY KEY,
	doctor_id int REFERENCES doctors(id) ON DELETE CASCADE,

	day_of_week int NOT NULL check(day_of_week in (1, 2, 3, 4, 5, 6, 7)), -- 1=Monday, 7=Sunday

	start_time time NOT NULL,
	end_time time NOT NULL,

	break_start time,
	break_end time
);

-- +goose Down
drop table normal_working_hours;
