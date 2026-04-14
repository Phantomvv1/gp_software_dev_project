package workinghours

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Phantomvv1/gp_software_dev_project/internal/auth"
	"github.com/gin-gonic/gin"
)

func setupRouter(repo WorkingHoursRepository, role byte, userID int) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	r.Use(func(c *gin.Context) {
		c.Set("id", userID)
		c.Set("role", role)
		c.Next()
	})

	r.GET("/working-hours/:doctor_id", GetWorkingHours(repo))
	r.POST("/working-hours/permanent", AddPermanentChange(repo))
	r.POST("/working-hours/override", AddOverride(repo))
	r.DELETE("/working-hours/override/:id", DeleteOverride(repo))

	return r
}

func TestGetWorkingHours(t *testing.T) {
	repo := NewTestRepository()
	r := setupRouter(repo, auth.Doctor, 1)

	req, _ := http.NewRequest("GET", "/working-hours/1", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatal("expected 200, got", w.Code)
	}
}

func TestAddPermanentChange_Success(t *testing.T) {
	repo := NewTestRepository()
	r := setupRouter(repo, auth.Doctor, 1)

	body := map[string]interface{}{
		"effective_from": time.Now().Add(8 * 24 * time.Hour),
		"hour": map[string]interface{}{
			"day_of_week": 1,
		},
	}

	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/working-hours/permanent", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("id", 1)
	c.Set("role", byte(auth.Doctor))

	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatal("expected 200, got", w.Code)
	}
}

func TestAddPermanentChange_Forbidden(t *testing.T) {
	repo := NewTestRepository()
	w := httptest.NewRecorder()

	body := `{}`
	req, _ := http.NewRequest("POST", "/working-hours/permanent", bytes.NewBufferString(body))

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("id", 1)
	c.Set("role", byte(auth.Patient))

	AddPermanentChange(repo)(c)

	if w.Code != 403 {
		t.Fatal("expected 403, got", w.Code)
	}
}

func TestAddOverride_Success(t *testing.T) {
	repo := NewTestRepository()
	w := httptest.NewRecorder()

	body := `{
	"start_date":"2026-01-01T10:00:00Z",
	"end_date":"2026-01-02T10:00:00Z"
	}`
	req, _ := http.NewRequest("POST", "/working-hours/override", bytes.NewBufferString(body))

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("id", 1)
	c.Set("role", byte(auth.Doctor))

	AddOverride(repo)(c)

	if w.Code != 200 {
		t.Fatal("expected 200, got", w.Code)
	}
}

func TestDeleteOverride_Success(t *testing.T) {
	repo := NewTestRepository()

	repo.Overrides[1] = WorkingHourOverride{
		ID:        1,
		DoctorID:  1,
		StartDate: time.Now(),
		EndDate:   time.Now().Add(24 * time.Hour),
	}

	repo.OverrideByID[1] = repo.Overrides[1]

	w := httptest.NewRecorder()

	req, _ := http.NewRequest("DELETE", "/working-hours/override/1", nil)

	c, _ := gin.CreateTestContext(w)
	c.Params = []gin.Param{{Key: "id", Value: "1"}}
	c.Request = req
	c.Set("id", 1)
	c.Set("role", byte(auth.Doctor))

	DeleteOverride(repo)(c)

	if w.Code != 200 {
		body, _ := io.ReadAll(w.Result().Body)
		log.Println(string(body))
	}
}

func TestDeleteOverride_Forbidden(t *testing.T) {
	repo := NewTestRepository()
	w := httptest.NewRecorder()

	req, _ := http.NewRequest("DELETE", "/working-hours/override/1", nil)

	c, _ := gin.CreateTestContext(w)
	c.Params = []gin.Param{{Key: "id", Value: "1"}}
	c.Request = req
	c.Set("id", 1)
	c.Set("role", byte(auth.Patient))

	DeleteOverride(repo)(c)

	if w.Code != 403 {
		t.Fatal("expected 403, got", w.Code)
	}
}
