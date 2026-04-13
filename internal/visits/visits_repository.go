package visits

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/Phantomvv1/gp_software_dev_project/internal/auth"
	endpointerrors "github.com/Phantomvv1/gp_software_dev_project/internal/endpoint_errors"
	"github.com/jackc/pgx/v5"
)

const (
	Doctor = iota + 1
	Patient
)

var (
	parsingError        = errors.New("Error: invalid id")
	dbConnectionError   = errors.New("Error: database connection failed")
	dbInsertVisitError  = errors.New("Error: unable to create visit")
	dbSelectVisitError  = errors.New("Error: unable to fetch visit")
	dbConflictError     = errors.New("Error: visit overlaps with another visit")
	dbDeleteVisitError  = errors.New("Error: unable to delete visit")
	dbSelectDoctorError = errors.New("Error: unable to get your doctor")
	tooSoonError        = errors.New("Error: visit must be created 24h in advance")
	cancelTooLateError  = errors.New("Error: cannot cancel less than 12h before")
)

type VisitsRepository interface {
	CreateVisit(visit Visit) (*Visit, error)
	CancelVisit(id string, userID int, role byte) error
	GetMyVisits(userID int, role byte) ([]*Visit, error)
}

type ProdRepository struct{}
type TestRepository struct{}

func (p ProdRepository) CreateVisit(v Visit) (*Visit, error) {
	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		return nil, endpointerrors.EndpointError{
			StatusCode: http.StatusInternalServerError,
			Err:        dbConnectionError,
		}
	}
	defer conn.Close(context.Background())

	err = conn.QueryRow(context.Background(),
		"select doctor_id from patients where id=$1",
		v.PatientID,
	).Scan(&v.DoctorID)

	if err != nil {
		return nil, endpointerrors.EndpointError{
			StatusCode: http.StatusBadRequest,
			Err:        dbSelectDoctorError,
		}
	}

	// 2. 24h rule
	if time.Until(v.StartTime) < 24*time.Hour {
		return nil, endpointerrors.EndpointError{
			StatusCode: http.StatusBadRequest,
			Err:        tooSoonError,
		}
	}

	// 3. Overlap check
	var exists bool
	err = conn.QueryRow(context.Background(),
		`select exists(
			select 1 from visits 
			where doctor_id=$1 
			and tstzrange(start_time, start_time + visit_time) && tstzrange($2,$3)
		)`,
		v.DoctorID, v.StartTime, v.EndTime,
	).Scan(&exists)

	if err != nil {
		return nil, endpointerrors.EndpointError{
			StatusCode: http.StatusInternalServerError,
			Err:        dbSelectVisitError,
		}
	}

	if exists {
		return nil, endpointerrors.EndpointError{
			StatusCode: http.StatusConflict,
			Err:        dbConflictError,
		}
	}

	// 4. Insert
	err = conn.QueryRow(context.Background(),
		`insert into visits (start_time, visit_time, patient_id, doctor_id)
		 values ($1, $2, $3, $4) returning id`,
		v.StartTime,
		v.EndTime.Sub(v.StartTime),
		v.PatientID,
		v.DoctorID,
	).Scan(&v.ID)

	if err != nil {
		log.Println(err)
		return nil, endpointerrors.EndpointError{
			StatusCode: http.StatusInternalServerError,
			Err:        dbInsertVisitError,
		}
	}

	return &v, nil
}

func (p ProdRepository) CancelVisit(idStr string, userID int, role byte) error {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return endpointerrors.EndpointError{
			StatusCode: http.StatusBadRequest,
			Err:        parsingError,
		}
	}

	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		return endpointerrors.EndpointError{
			StatusCode: http.StatusInternalServerError,
			Err:        dbConnectionError,
		}
	}
	defer conn.Close(context.Background())

	visit := Visit{}
	err = conn.QueryRow(context.Background(),
		`select id, start_time, patient_id, doctor_id 
		 from visits where id=$1`,
		id,
	).Scan(&visit.ID, &visit.StartTime, &visit.PatientID, &visit.DoctorID)

	if err != nil {
		return endpointerrors.EndpointError{
			StatusCode: http.StatusInternalServerError,
			Err:        dbSelectVisitError,
		}
	}

	if role == auth.Patient && visit.PatientID != userID {
		return endpointerrors.EndpointError{
			StatusCode: http.StatusForbidden,
			Err:        errors.New("Error: not your visit"),
		}
	}

	if role == auth.Doctor && visit.DoctorID != userID {
		return endpointerrors.EndpointError{
			StatusCode: http.StatusForbidden,
			Err:        errors.New("Error: not your visit"),
		}
	}

	// 12h rule
	if time.Until(visit.StartTime) < 12*time.Hour {
		return endpointerrors.EndpointError{
			StatusCode: http.StatusBadRequest,
			Err:        cancelTooLateError,
		}
	}

	_, err = conn.Exec(context.Background(),
		"delete from visits where id=$1", id,
	)

	if err != nil {
		return endpointerrors.EndpointError{
			StatusCode: http.StatusInternalServerError,
			Err:        dbDeleteVisitError,
		}
	}

	return nil
}

func (p ProdRepository) GetMyVisits(userID int, role byte) ([]*Visit, error) {
	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		return nil, endpointerrors.EndpointError{
			StatusCode: http.StatusInternalServerError,
			Err:        dbConnectionError,
		}
	}
	defer conn.Close(context.Background())

	query := ""
	if role == auth.Patient {
		query = "select id, start_time, visit_time, patient_id, doctor_id from visits where patient_id=$1"
	} else {
		query = "select id, start_time, visit_time, patient_id, doctor_id from visits where doctor_id=$1"
	}

	rows, err := conn.Query(context.Background(), query, userID)
	if err != nil {
		return nil, endpointerrors.EndpointError{
			StatusCode: http.StatusInternalServerError,
			Err:        dbSelectVisitError,
		}
	}

	visits, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (*Visit, error) {
		v := &Visit{}
		var duration time.Duration

		err := row.Scan(&v.ID, &v.StartTime, &duration, &v.PatientID, &v.DoctorID)
		v.EndTime = v.StartTime.Add(duration)

		return v, err
	})

	if err != nil {
		return nil, endpointerrors.EndpointError{
			StatusCode: http.StatusInternalServerError,
			Err:        dbSelectVisitError,
		}
	}

	return visits, nil
}
