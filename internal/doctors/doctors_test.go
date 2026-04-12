package doctors

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

	doc := r.Group("/doctors")
	doc.GET("/:id", GetDoctorById(repo))
	doc.GET("", GetAllDoctors(repo))
	doc.POST("", RegisterDoctor(repo))
	doc.PUT("/:id", UpdateDoctor(repo))
	doc.DELETE("/:id", DeleteDoctor(repo))

	return r
}

func TestRegisterDoctor_Success(t *testing.T) {
	r := setupRouter()

	body := Doctor{
		Name:    "Test Doctor",
		Email:   "test@email.com",
		Address: "Sofia",
	}

	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPost, "/doctors", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", w.Code)
	}
}

func TestRegisterDoctor_BadRequest(t *testing.T) {
	r := setupRouter()

	req, _ := http.NewRequest(http.MethodPost, "/doctors", bytes.NewBuffer([]byte(`invalid json`)))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetAllDoctors(t *testing.T) {
	r := setupRouter()

	req, _ := http.NewRequest(http.MethodGet, "/doctors", nil)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestGetDoctorById(t *testing.T) {
	r := setupRouter()

	req, _ := http.NewRequest(http.MethodGet, "/doctors/1", nil)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestUpdateDoctor_Success(t *testing.T) {
	r := setupRouter()

	body := Doctor{
		Name:    "Updated Doctor",
		Email:   "updated@email.com",
		Address: "Plovdiv",
	}

	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPut, "/doctors/1", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestUpdateDoctor_BadRequest(t *testing.T) {
	r := setupRouter()

	req, _ := http.NewRequest(http.MethodPut, "/doctors/1", bytes.NewBuffer([]byte(`bad json`)))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestDeleteDoctor(t *testing.T) {
	r := setupRouter()

	req, _ := http.NewRequest(http.MethodDelete, "/doctors/1", nil)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestGetAllDoctors_WithLimit(t *testing.T) {
	r := setupRouter()

	req, _ := http.NewRequest(http.MethodGet, "/doctors?limit=1", nil)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}
