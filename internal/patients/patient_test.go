package patients

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	repo := TestRepository{}

	p := r.Group("/patients")
	{
		p.POST("", RegisterPatient(repo))
		p.GET("", GetAllPatients(repo))
		p.GET("/:id", GetPatientById(repo))
		p.PUT("/:id", UpdatePatient(repo))
		p.DELETE("/:id", DeletePatient(repo))
	}

	return r
}

func TestRegisterPatient_Success(t *testing.T) {
	r := setupRouter()

	body := Patient{
		Name:        "John Doe",
		Email:       "john@mail.com",
		PhoneNumber: "123456",
	}

	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPost, "/patients", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatal("expected status 201, got", w.Code)
	}

	if !bytes.Contains(w.Body.Bytes(), []byte("John Doe")) {
		t.Fatal("expected response to contain patient name")
	}
}

func TestRegisterPatient_BadJSON(t *testing.T) {
	r := setupRouter()

	req, _ := http.NewRequest(http.MethodPost, "/patients", bytes.NewBuffer([]byte("{bad json")))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatal("expected status 400 for bad JSON, got", w.Code)
	}
}

func TestGetAllPatients_Default(t *testing.T) {
	r := setupRouter()

	req, _ := http.NewRequest(http.MethodGet, "/patients", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatal("expected status 200, got", w.Code)
	}

	if !bytes.Contains(w.Body.Bytes(), []byte("Patient1")) {
		t.Fatal("expected response to contain Patient1")
	}
}

func TestGetAllPatients_WithLimit(t *testing.T) {
	r := setupRouter()

	req, _ := http.NewRequest(http.MethodGet, "/patients?limit=1", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatal("expected status 200 with limit, got", w.Code)
	}
}

func TestGetPatientById_Success(t *testing.T) {
	r := setupRouter()

	req, _ := http.NewRequest(http.MethodGet, "/patients/1", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatal("expected status 200, got", w.Code)
	}

	if !bytes.Contains(w.Body.Bytes(), []byte("Test Patient")) {
		t.Fatal("expected response to contain Test Patient")
	}
}

func TestGetPatientById_InvalidID(t *testing.T) {
	r := setupRouter()

	req, _ := http.NewRequest(http.MethodGet, "/patients/abc", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatal("expected status 200 even for invalid id in test repo, got", w.Code)
	}
}

func TestUpdatePatient_Success(t *testing.T) {
	r := setupRouter()

	body := Patient{
		Name:        "Updated Name",
		Email:       "updated@mail.com",
		PhoneNumber: "999",
	}

	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPut, "/patients/1", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatal("expected status 200, got", w.Code)
	}

	if !bytes.Contains(w.Body.Bytes(), []byte("Updated Name")) {
		t.Fatal("expected updated name in response")
	}
}

func TestUpdatePatient_BadJSON(t *testing.T) {
	r := setupRouter()

	req, _ := http.NewRequest(http.MethodPut, "/patients/1", bytes.NewBuffer([]byte("{bad json")))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatal("expected status 400 for bad JSON, got", w.Code)
	}
}

func TestDeletePatient_Success(t *testing.T) {
	r := setupRouter()

	req, _ := http.NewRequest(http.MethodDelete, "/patients/1", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatal("expected status 200, got", w.Code)
	}

	if !bytes.Contains(w.Body.Bytes(), []byte("patient deleted")) {
		t.Fatal("expected delete confirmation message")
	}
}
