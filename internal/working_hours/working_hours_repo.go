package workinghours

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	endpointerrors "github.com/Phantomvv1/gp_software_dev_project/internal/endpoint_errors"
	"github.com/jackc/pgx/v5"
)

var (
	dbConnectionError            = errors.New("Error: unable to connect to the database")
	dbInsertWorkingHoursError    = errors.New("Error: unable to insert working hours")
	dbDeleteWorkingHoursError    = errors.New("Error: unable to delete working hours")
	dbInsertOverrideError        = errors.New("Error: unable to insert override")
	dbExistingOverrideError      = errors.New("Error: override already exists")
	dbSelectDoctorError          = errors.New("Error: unable to get the doctor from the database")
	dbInsertPermanentChangeError = errors.New("Error: unable to insert permanent change")
	invalidFutureChangeError     = errors.New("Error: permanent change must be at least 7 days in future")
)

type WorkingHoursRepository interface {
	SetWorkingHours(doctorID int, hours []WorkingHour) error
	AddOverride(override WorkingHourOverride) error
	AddPermanentChange(doctorID int, effectiveFrom time.Time, hours []WorkingHour) error
}

type ProdRepository struct{}
type TestRepository struct{}

func (p ProdRepository) SetWorkingHours(doctorID int, hours []WorkingHour) error {
	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		return endpointerrors.EndpointError{
			StatusCode: http.StatusInternalServerError,
			Err:        dbConnectionError,
		}
	}
	defer conn.Close(context.Background())

	tx, err := conn.Begin(context.Background())
	if err != nil {
		return endpointerrors.EndpointError{
			StatusCode: 500,
			Err:        dbConnectionError,
		}
	}
	defer tx.Rollback(context.Background())

	// delete old schedule
	_, err = tx.Exec(context.Background(),
		"delete from doctor_working_hours where doctor_id=$1",
		doctorID,
	)
	if err != nil {
		return endpointerrors.EndpointError{
			StatusCode: 500,
			Err:        dbDeleteWorkingHoursError,
		}
	}

	// insert new
	for _, h := range hours {
		_, err = tx.Exec(context.Background(),
			`insert into doctor_working_hours 
			(doctor_id, day_of_week, start_time, end_time, break_start, break_end)
			values ($1,$2,$3,$4,$5,$6)`,
			doctorID, h.DayOfWeek, h.StartTime, h.EndTime, h.BreakStart, h.BreakEnd,
		)

		if err != nil {
			return endpointerrors.EndpointError{
				StatusCode: 500,
				Err:        dbInsertWorkingHoursError,
			}
		}
	}

	return tx.Commit(context.Background())
}

func (p ProdRepository) AddOverride(o WorkingHourOverride) error {
	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		return endpointerrors.EndpointError{
			StatusCode: 500,
			Err:        dbConnectionError,
		}
	}
	defer conn.Close(context.Background())

	// only ONE active override allowed
	exists := false
	err = conn.QueryRow(context.Background(),
		`select exists(
			select 1 from doctor_working_hours_overrides 
			where doctor_id=$1
		)`,
		o.DoctorID,
	).Scan(&exists)

	if err != nil {
		return endpointerrors.EndpointError{
			StatusCode: 500,
			Err:        dbSelectDoctorError,
		}
	}

	if exists {
		return endpointerrors.EndpointError{
			StatusCode: 400,
			Err:        dbExistingOverrideError,
		}
	}

	_, err = conn.Exec(context.Background(),
		`insert into doctor_working_hours_overrides
		(doctor_id, start_date, end_date, start_time, end_time, break_start, break_end)
		values ($1,$2,$3,$4,$5,$6,$7)`,
		o.DoctorID, o.StartDate, o.EndDate,
		o.StartTime, o.EndTime, o.BreakStart, o.BreakEnd,
	)

	if err != nil {
		return endpointerrors.EndpointError{
			StatusCode: 500,
			Err:        dbInsertOverrideError,
		}
	}

	return nil
}

func (p ProdRepository) AddPermanentChange(doctorID int, effectiveFrom time.Time, hour WorkingHour) error {
	// ✅ rule: at least 7 days ahead
	if time.Until(effectiveFrom) < 7*24*time.Hour {
		return endpointerrors.EndpointError{
			StatusCode: 400,
			Err:        errors.New("permanent changes must be at least 7 days in advance"),
		}
	}

	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		return endpointerrors.EndpointError{
			StatusCode: 500,
			Err:        dbConnectionError,
		}
	}
	defer conn.Close(context.Background())

	err = insertWorkingHourVersion(conn, doctorID, effectiveFrom, hour)
	if err != nil {
		return endpointerrors.EndpointError{
			StatusCode: 500,
			Err:        err,
		}
	}

	return nil
}

func insertWorkingHourVersion(conn *pgx.Conn, doctorID int, effectiveFrom time.Time, hour WorkingHour) error {
	_, err := conn.Exec(context.Background(),
		`insert into normal_working_hours
		(doctor_id, day_of_week, start_time, end_time, break_start, break_end, effective_from)
		values ($1,$2,$3,$4,$5,$6,$7)`,
		doctorID,
		hour.DayOfWeek,
		hour.StartTime,
		hour.EndTime,
		hour.BreakStart,
		hour.BreakEnd,
		effectiveFrom,
	)

	if err != nil {
		log.Println(err)
		return dbInsertWorkingHoursError
	}

	return nil
}
