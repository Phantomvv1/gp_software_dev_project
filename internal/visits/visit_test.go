package visits

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Phantomvv1/gp_software_dev_project/internal/auth"
	"github.com/gin-gonic/gin"
)

type testRepo struct{}

func (t testRepo) CreateVisit(v Visit) (*Visit, error) {
	v.ID = 1
	return &v, nil
}

func (t testRepo) CancelVisit(id string, userID int, role byte) error {
	return nil
}

func (t testRepo) GetMyVisits(userID int, role byte) ([]*Visit, error) {
	return []*Visit{
		{ID: 1, PatientID: userID},
	}, nil
}

func TestCreateVisit_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := testRepo{}

	body := Visit{
		StartTime: time.Now().Add(48 * time.Hour),
		EndTime:   time.Now().Add(49 * time.Hour),
		DoctorID:  1,
	}

	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/visits", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("id", 1)
	c.Set("role", byte(auth.Patient))

	CreateVisit(repo)(c)

	if w.Code != 201 {
		t.Fatal("expected 201, got", w.Code)
	}
}

func TestCreateVisit_Forbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := testRepo{}

	req, _ := http.NewRequest("POST", "/visits", bytes.NewBufferString(`{}`))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("id", 1)
	c.Set("role", byte(auth.Doctor))

	CreateVisit(repo)(c)

	if w.Code != 403 {
		t.Fatal("expected 403, got", w.Code)
	}
}

func TestCancelVisit_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := testRepo{}

	req, _ := http.NewRequest("DELETE", "/visits/1", nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = []gin.Param{{Key: "id", Value: "1"}}
	c.Request = req
	c.Set("id", 1)
	c.Set("role", byte(auth.Patient))

	CancelVisit(repo)(c)

	if w.Code != 200 {
		t.Fatal("expected 200, got", w.Code)
	}
}

func TestGetMyVisits_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := testRepo{}

	req, _ := http.NewRequest("GET", "/visits/me", nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("id", 1)
	c.Set("role", byte(auth.Patient))

	GetMyVisits(repo)(c)

	if w.Code != 200 {
		t.Fatal("expected 200, got", w.Code)
	}
}
