package workinghours

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	endpointerrors "github.com/Phantomvv1/gp_software_dev_project/internal/endpoint_errors"
	"github.com/jackc/pgx/v5"
)

var (
	parsingError                 = errors.New("Error: unable to parse the id of the doctor")
	dbConnectionError            = errors.New("Error: unable to connect to the database")
	dbInsertWorkingHoursError    = errors.New("Error: unable to insert working hours")
	dbDeleteWorkingHoursError    = errors.New("Error: unable to delete working hours")
	dbInsertOverrideError        = errors.New("Error: unable to insert override")
	dbExistingOverrideError      = errors.New("Error: override already exists")
	dbSelectDoctorError          = errors.New("Error: unable to get the doctor from the database")
	dbSelectWorkingHoursError    = errors.New("Error: unable to get the working hours for the doctor from the database")
	dbInsertPermanentChangeError = errors.New("Error: unable to insert permanent change")
	invalidFutureChangeError     = errors.New("Error: permanent change must be at least 7 days in future")
)

type WorkingHoursRepository interface {
	GetWorkingHours(doctorID int, date time.Time) ([]*WorkingHour, error)
	AddPermanentChange(doctorID int, effectiveFrom time.Time, hour WorkingHour) error
	SetWorkingHours(doctorID int, hours []WorkingHour) error
	AddOverride(override WorkingHourOverride) error
	DeleteOverride(id string, doctorID int) error
}

type ProdRepository struct{}
type TestRepository struct {
	WorkingHours map[int][]*WorkingHour      // doctorID -> hours
	Overrides    map[int]WorkingHourOverride // doctorID -> override (only one allowed)
	OverrideByID map[int]WorkingHourOverride // overrideID -> override
	NextID       int
}

func NewTestRepository() *TestRepository {
	return &TestRepository{
		WorkingHours: make(map[int][]*WorkingHour),
		Overrides:    make(map[int]WorkingHourOverride),
		OverrideByID: make(map[int]WorkingHourOverride),
		NextID:       1,
	}
}

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

func (p ProdRepository) GetWorkingHours(doctorID int, date time.Time) ([]*WorkingHour, error) {
	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		return nil, endpointerrors.EndpointError{StatusCode: 500, Err: dbConnectionError}
	}
	defer conn.Close(context.Background())

	rows, err := conn.Query(context.Background(),
		`
		select distinct on (day_of_week)
			day_of_week, start_time, end_time, break_start, break_end
		from normal_working_hours
		where doctor_id=$1 and effective_from <= $2
		order by day_of_week, effective_from desc
		`,
		doctorID,
		date,
	)

	if err != nil {
		return nil, endpointerrors.EndpointError{StatusCode: 500, Err: dbSelectWorkingHoursError}
	}

	hours, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (*WorkingHour, error) {
		h := &WorkingHour{}
		err := row.Scan(&h.DayOfWeek, &h.StartTime, &h.EndTime, &h.BreakStart, &h.BreakEnd)
		return h, err
	})

	if err != nil {
		return nil, endpointerrors.EndpointError{StatusCode: 500, Err: dbSelectWorkingHoursError}
	}

	return hours, nil
}

func (p ProdRepository) DeleteOverride(idStr string, doctorID int) error {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return endpointerrors.EndpointError{StatusCode: 400, Err: parsingError}
	}

	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		return endpointerrors.EndpointError{StatusCode: 500, Err: dbConnectionError}
	}
	defer conn.Close(context.Background())

	cmd, err := conn.Exec(context.Background(),
		`delete from doctor_working_overrides 
		 where id=$1 and doctor_id=$2`,
		id, doctorID,
	)

	if err != nil {
		return endpointerrors.EndpointError{StatusCode: 500, Err: dbDeleteWorkingHoursError}
	}

	if cmd.RowsAffected() == 0 {
		return endpointerrors.EndpointError{StatusCode: 404, Err: errors.New("override not found")}
	}

	return nil
}

func (t TestRepository) GetWorkingHours(doctorID int, date time.Time) ([]*WorkingHour, error) {
	hours, ok := t.WorkingHours[doctorID]
	if !ok {
		return []*WorkingHour{}, nil
	}
	return hours, nil
}

func (t TestRepository) AddPermanentChange(
	doctorID int,
	effectiveFrom time.Time,
	hour WorkingHour,
) error {

	if time.Until(effectiveFrom) < 7*24*time.Hour {
		return endpointerrors.EndpointError{
			StatusCode: 400,
			Err:        errors.New("permanent changes must be at least 7 days in advance"),
		}
	}

	t.WorkingHours[doctorID] = append(t.WorkingHours[doctorID], &hour)

	return nil
}

func (t TestRepository) AddOverride(o WorkingHourOverride) error {
	// only one override allowed
	if _, exists := t.Overrides[o.DoctorID]; exists {
		return endpointerrors.EndpointError{
			StatusCode: 400,
			Err:        dbExistingOverrideError,
		}
	}

	o.ID = t.NextID
	t.NextID++

	t.Overrides[o.DoctorID] = o
	t.OverrideByID[o.ID] = o

	return nil
}

func (t TestRepository) DeleteOverride(idStr string, doctorID int) error {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return endpointerrors.EndpointError{
			StatusCode: 400,
			Err:        parsingError,
		}
	}

	override, exists := t.OverrideByID[id]
	if !exists {
		return endpointerrors.EndpointError{
			StatusCode: 404,
			Err:        errors.New("override not found"),
		}
	}

	if override.DoctorID != doctorID {
		return endpointerrors.EndpointError{
			StatusCode: 404,
			Err:        errors.New("override not found"),
		}
	}

	delete(t.OverrideByID, id)
	delete(t.Overrides, override.DoctorID)

	return nil
}

func (t TestRepository) SetWorkingHours(doctorID int, hours []WorkingHour) error {
	var res []*WorkingHour

	for _, h := range hours {
		hCopy := h
		hCopy.DoctorID = doctorID
		res = append(res, &hCopy)
	}

	t.WorkingHours[doctorID] = res
	return nil
}
