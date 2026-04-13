package visits

import (
	"net/http"
	"time"

	"github.com/Phantomvv1/gp_software_dev_project/internal/auth"
	endpointerrors "github.com/Phantomvv1/gp_software_dev_project/internal/endpoint_errors"
	"github.com/gin-gonic/gin"
)

type Visit struct {
	ID        int       `json:"id"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	PatientID int       `json:"patient_id"`
	DoctorID  int       `json:"doctor_id"`
}

func CreateVisit(repo VisitsRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		v := Visit{}

		if err := c.ShouldBindJSON(&v); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		userID := c.GetInt("id")
		roleAny, _ := c.Get("role")
		role, _ := roleAny.(byte)

		if byte(role) != auth.Patient {
			c.JSON(http.StatusForbidden, gin.H{"error": "Error: only patients can create visits"})
			return
		}

		v.PatientID = userID

		created, err := repo.CreateVisit(v)
		if err != nil {
			err := err.(endpointerrors.EndpointError)
			c.JSON(err.StatusCode, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"result": created})
	}
}

func CancelVisit(repo VisitsRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		userID := c.GetInt("id")
		roleAny, _ := c.Get("role")
		role := roleAny.(byte)

		err := repo.CancelVisit(id, userID, role)
		if err != nil {
			err := err.(endpointerrors.EndpointError)
			c.JSON(err.StatusCode, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"result": "visit cancelled"})
	}
}

func GetMyVisits(repo VisitsRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt("id")
		roleAny, _ := c.Get("role")
		role := roleAny.(byte)

		visits, err := repo.GetMyVisits(userID, role)
		if err != nil {
			err := err.(endpointerrors.EndpointError)
			c.JSON(err.StatusCode, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"result": visits})
	}
}
