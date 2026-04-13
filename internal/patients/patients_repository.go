package patients

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/Phantomvv1/gp_software_dev_project/internal/auth"
	endpointerrors "github.com/Phantomvv1/gp_software_dev_project/internal/endpoint_errors"
	"github.com/jackc/pgx/v5"
)

var (
	parsingError                = errors.New("Error: unable to parse the id of the patient")
	dbConnectionError           = errors.New("Error: unable to connect to the database")
	dbInsertPatientError        = errors.New("Error: unable to insert patient into database")
	dbSelectPatientError        = errors.New("Error: unable to get patient from database")
	dbSelectAllPatientsQueryErr = errors.New("Error: query error while getting patients")
	dbSelectAllPatientsScanErr  = errors.New("Error: scan error while collecting patients")
	dbUpdatePatientError        = errors.New("Error: unable to update patient")
	dbDeletePatientError        = errors.New("Error: unable to delete patient")
)

type PatientsRepository interface {
	Register(patient Patient) (*Patient, error)
	GetAllPatients(limit string) ([]*Patient, error)
	GetPatientById(id string) (*Patient, error)
	UpdatePatient(id string, userId int, patient Patient) (*Patient, error)
	DeletePatient(id string) error
}

type ProdRepository struct{}
type TestRepository struct{}

func (p ProdRepository) Register(patient Patient) (*Patient, error) {
	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		return nil, endpointerrors.EndpointError{
			StatusCode: 500,
			Err:        dbConnectionError,
		}
	}
	defer conn.Close(context.Background())

	patient.Password = auth.SHA512(patient.Password)

	registeredPatient, err := insertPatient(conn, patient)
	if err != nil {
		return nil, endpointerrors.EndpointError{
			StatusCode: 500,
			Err:        err,
		}
	}

	return registeredPatient, nil
}

func insertPatient(conn *pgx.Conn, patient Patient) (*Patient, error) {
	err := conn.QueryRow(context.Background(),
		`insert into patients (name, email, phone_number, doctor_id, password)
		 values ($1,$2,$3,$4,$5) returning id`,
		patient.Name, patient.Email, patient.PhoneNumber, patient.DoctorID, patient.Password,
	).Scan(&patient.ID)

	if err != nil {
		log.Println(err)
		return nil, dbInsertPatientError
	}

	return &patient, nil
}

func (p ProdRepository) GetAllPatients(limit string) ([]*Patient, error) {
	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		return nil, endpointerrors.EndpointError{
			StatusCode: 500,
			Err:        dbConnectionError,
		}
	}
	defer conn.Close(context.Background())

	return getAllPatients(conn, limit)
}

func getAllPatients(conn *pgx.Conn, limit string) ([]*Patient, error) {
	query := "select id, name, email, phone_number, doctor_id from patients limit "
	if limit != "" && limit != "0" {
		query += limit
	} else {
		query += "20"
	}

	rows, err := conn.Query(context.Background(), query)
	if err != nil {
		log.Println(err)
		return nil, dbSelectAllPatientsQueryErr
	}

	patients, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (*Patient, error) {
		p := &Patient{}
		err := row.Scan(&p.ID, &p.Name, &p.Email, &p.PhoneNumber, &p.DoctorID)
		return p, err
	})

	if err != nil {
		log.Println(err)
		return nil, dbSelectAllPatientsScanErr
	}

	return patients, nil
}

func (p ProdRepository) GetPatientById(idStr string) (*Patient, error) {
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return nil, endpointerrors.EndpointError{
			StatusCode: 400,
			Err:        parsingError,
		}
	}

	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		return nil, endpointerrors.EndpointError{
			StatusCode: 500,
			Err:        dbConnectionError,
		}
	}
	defer conn.Close(context.Background())

	return getPatient(conn, int(id))
}

func getPatient(conn *pgx.Conn, id int) (*Patient, error) {
	p := Patient{}

	err := conn.QueryRow(context.Background(),
		`select id, name, email, phone_number, doctor_id 
		 from patients where id=$1`,
		id,
	).Scan(&p.ID, &p.Name, &p.Email, &p.PhoneNumber, &p.DoctorID)

	if err != nil {
		log.Println(err)
		return nil, dbSelectPatientError
	}

	return &p, nil
}

func (p ProdRepository) UpdatePatient(idStr string, userId int, patient Patient) (*Patient, error) {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return nil, endpointerrors.EndpointError{
			StatusCode: 400,
			Err:        parsingError,
		}
	}

	if id != userId {
		return nil, endpointerrors.EndpointError{
			StatusCode: http.StatusForbidden,
			Err:        errors.New("Error: you are allowed to update only you profile"),
		}
	}

	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		return nil, endpointerrors.EndpointError{
			StatusCode: 500,
			Err:        dbConnectionError,
		}
	}
	defer conn.Close(context.Background())

	err = updatePatient(conn, id, patient)
	if err != nil {
		return nil, endpointerrors.EndpointError{
			StatusCode: 500,
			Err:        err,
		}
	}

	patient.ID = int(id)
	return &patient, nil
}

func updatePatient(conn *pgx.Conn, id int, patient Patient) error {
	_, err := conn.Exec(context.Background(),
		`update patients 
		 set name=$1, email=$2, phone_number=$3, doctor_id=$4
		 where id=$5`,
		patient.Name, patient.Email, patient.PhoneNumber, patient.DoctorID, id,
	)

	if err != nil {
		log.Println(err)
		return dbUpdatePatientError
	}

	return nil
}

func (p ProdRepository) DeletePatient(idStr string) error {
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return endpointerrors.EndpointError{
			StatusCode: 400,
			Err:        parsingError,
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

	err = deletePatient(conn, int(id))
	if err != nil {
		return endpointerrors.EndpointError{
			StatusCode: 500,
			Err:        err,
		}
	}

	return nil
}

func deletePatient(conn *pgx.Conn, id int) error {
	_, err := conn.Exec(context.Background(),
		"delete from patients where id=$1", id)

	if err != nil {
		log.Println(err)
		return dbDeletePatientError
	}

	return nil
}

func (t TestRepository) Register(patient Patient) (*Patient, error) {
	patient.ID = 1
	return &patient, nil
}

func (t TestRepository) GetAllPatients(limit string) ([]*Patient, error) {
	return []*Patient{
		{ID: 1, Name: "Patient1", Email: "p1@mail.com"},
		{ID: 2, Name: "Patient2", Email: "p2@mail.com"},
	}, nil
}

func (t TestRepository) GetPatientById(id string) (*Patient, error) {
	return &Patient{ID: 1, Name: "Test Patient"}, nil
}

func (t TestRepository) UpdatePatient(id string, userId int, patient Patient) (*Patient, error) {
	return &patient, nil
}

func (t TestRepository) DeletePatient(id string) error {
	return nil
}
