package auth

import (
	"context"
	"net/http"
	"os"

	endpointerrors "github.com/Phantomvv1/gp_software_dev_project/internal/endpoint_errors"
	"github.com/jackc/pgx/v5"
)

type authRepository interface {
	Login(email, password string) (int, string, byte, error)
}

type ProdRepository struct{}
type TestRepository struct{}

func (p ProdRepository) Login(email, password string) (int, string, byte, error) {
	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		return 0, "", 0, endpointerrors.EndpointError{
			StatusCode: http.StatusInternalServerError,
			Err:        dbConnectionError,
		}
	}
	defer conn.Close(context.Background())

	hashed := SHA512(password)

	// 1. Doctors
	var id int
	var dbEmail string

	err = conn.QueryRow(context.Background(),
		"select id, email from doctors where email=$1 and password=$2",
		email, hashed,
	).Scan(&id, &dbEmail)

	if err == nil {
		return id, dbEmail, Doctor, nil
	}

	// 2. Patients
	err = conn.QueryRow(context.Background(),
		"select id, email from patients where email=$1 and password=$2",
		email, hashed,
	).Scan(&id, &dbEmail)

	if err == nil {
		return id, dbEmail, Patient, nil
	}

	return 0, "", 0, endpointerrors.EndpointError{
		StatusCode: http.StatusUnauthorized,
		Err:        invalidCredentialsError,
	}
}

func (t TestRepository) Login(email, password string) (string, error) {
	return GenerateJWT(1, Patient, "fake@email.com")
}
