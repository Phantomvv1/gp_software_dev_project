package doctors

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"

	endpointerrors "github.com/Phantomvv1/gp_software_dev_project/internal/endpoint_errors"
	"github.com/jackc/pgx/v5"
)

var (
	parsingError        = errors.New("Error: unable to parse the id of the doctor")
	dbConnectionError   = errors.New("Error: unable to connect to the database")
	dbSelectDoctorError = errors.New("Error: unable to get the doctor from the database")
	dbInsertDoctorError = errors.New("Error: unable to insert the information of the doctor into the database")
	dbUpdateDoctorError = errors.New("Error: unable to update the doctor information in the database")
)

type doctorsRepository interface {
	Register(doctor Doctor) (*Doctor, error)
	GetDoctorById(id string) (*Doctor, error)
	UpdateDoctor(id string, doctor Doctor) (*Doctor, error)
	DeleteDoctor(id string) error
}

type ProdRepository struct{}
type TestRepository struct{}

func (p ProdRepository) Register(doctor Doctor) (*Doctor, error) {
	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		return nil, endpointerrors.EndpointError{
			StatusCode: http.StatusInternalServerError,
			Err:        dbConnectionError,
		}
	}
	defer conn.Close(context.Background())

	registeredDoctor, err := insertDoctorIntoDB(conn, doctor)
	if err != nil {
		return nil, endpointerrors.EndpointError{
			StatusCode: http.StatusInternalServerError,
			Err:        err,
		}
	}

	return registeredDoctor, nil
}

func insertDoctorIntoDB(conn *pgx.Conn, doctor Doctor) (*Doctor, error) {
	err := conn.QueryRow(context.Background(),
		"insert into doctors (name, email, address) values ($1, $2, $3) returning id",
		doctor.Name, doctor.Email, doctor.Address,
	).Scan(&doctor.ID)
	if err != nil {
		log.Println(err)
		return nil, dbInsertDoctorError
	}

	return &doctor, nil
}

func (p ProdRepository) GetDoctorById(idStr string) (*Doctor, error) {
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return nil, endpointerrors.EndpointError{
			StatusCode: http.StatusBadRequest,
			Err:        parsingError,
		}
	}

	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		return nil, endpointerrors.EndpointError{
			StatusCode: http.StatusInternalServerError,
			Err:        dbConnectionError,
		}
	}
	defer conn.Close(context.Background())

	doctor, err := getDoctorFromDB(conn, int(id))
	if err != nil {
		return nil, endpointerrors.EndpointError{
			StatusCode: http.StatusInternalServerError,
			Err:        err,
		}
	}

	return doctor, nil
}

func getDoctorFromDB(conn *pgx.Conn, id int) (*Doctor, error) {
	doctor := Doctor{}
	err := conn.QueryRow(context.Background(), "select id, name, email, address, working_hours_id from doctors where id = $1", id).
		Scan(&doctor.ID, &doctor.Name, &doctor.Email, &doctor.Address, &doctor.WorkingHoursID)
	if err != nil {
		log.Println(err)
		return nil, dbSelectDoctorError
	}

	return &doctor, nil
}

func (p ProdRepository) UpdateDoctor(idStr string, doctor Doctor) (*Doctor, error) {
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return nil, endpointerrors.EndpointError{
			StatusCode: http.StatusBadRequest,
			Err:        parsingError,
		}
	}

	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		return nil, endpointerrors.EndpointError{
			StatusCode: http.StatusInternalServerError,
			Err:        dbConnectionError,
		}
	}
	defer conn.Close(context.Background())

	err = updateDoctor(conn, doctor, int(id))
	if err != nil {
		return nil, endpointerrors.EndpointError{
			StatusCode: http.StatusInternalServerError,
			Err:        err,
		}
	}

	return &doctor, nil
}

func updateDoctor(conn *pgx.Conn, doctor Doctor, id int) error {
	_, err := conn.Exec(context.Background(),
		"update doctors set name=$1, email=$2, address=$3 where id=$4",
		doctor.Name, doctor.Email, doctor.Address, id,
	)
	if err != nil {
		log.Println(err)
		return dbUpdateDoctorError
	}

	return nil
}

func (p ProdRepository) DeleteDoctor(idStr string) error {
	id, err := strconv.ParseInt(idStr, 10, 64)
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

	_, err = conn.Exec(context.Background(),
		"delete from doctors where id=$1",
		id,
	)

	if err != nil {
		return endpointerrors.EndpointError{
			StatusCode: http.StatusInternalServerError,
			Err:        err,
		}
	}

	return nil
}

func (t TestRepository) Register(doctor Doctor) (*Doctor, error) {
	doctor.ID = 1
	return &doctor, nil
}

func (t TestRepository) GetDoctorById(email string) (*Doctor, error) {
	return &Doctor{
		ID:      1,
		Name:    "Test Doctor",
		Email:   email,
		Address: "Test Address",
	}, nil
}

func (t TestRepository) UpdateDoctor(id string, doctor Doctor) (*Doctor, error) {
	return &doctor, nil
}

func (t TestRepository) DeleteDoctor(id string) error {
	return nil
}
