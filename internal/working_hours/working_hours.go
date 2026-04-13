package workinghours

import (
	"time"

	"github.com/Phantomvv1/gp_software_dev_project/internal/auth"
	endpointerrors "github.com/Phantomvv1/gp_software_dev_project/internal/endpoint_errors"
	"github.com/gin-gonic/gin"
)

type WorkingHour struct {
	ID         int    `json:"id"`
	DoctorID   int    `json:"doctor_id"`
	DayOfWeek  int    `json:"day_of_week"`
	StartTime  string `json:"start_time"`
	EndTime    string `json:"end_time"`
	BreakStart string `json:"break_start,omitempty"`
	BreakEnd   string `json:"break_end,omitempty"`
}

type WorkingHourOverride struct {
	ID         int       `json:"id"`
	DoctorID   int       `json:"doctor_id"`
	StartDate  time.Time `json:"start_date"`
	EndDate    time.Time `json:"end_date"`
	StartTime  *string   `json:"start_time,omitempty"`
	EndTime    *string   `json:"end_time,omitempty"`
	BreakStart *string   `json:"break_start,omitempty"`
	BreakEnd   *string   `json:"break_end,omitempty"`
}

func SetWorkingHours(repo WorkingHoursRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var hours []WorkingHour

		if err := c.ShouldBindJSON(&hours); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		roleAny, _ := c.Get("role")
		role := roleAny.(byte)
		doctorID := c.GetInt("id")

		if role != auth.Doctor {
			c.JSON(403, gin.H{"error": "only doctors allowed"})
			return
		}

		err := repo.SetWorkingHours(doctorID, hours)
		if err != nil {
			err := err.(endpointerrors.EndpointError)
			c.JSON(err.StatusCode, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"result": "working hours updated"})
	}
}

func AddOverride(repo WorkingHoursRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		o := WorkingHourOverride{}

		if err := c.ShouldBindJSON(&o); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		roleAny, _ := c.Get("role")
		role := roleAny.(byte)
		o.DoctorID = c.GetInt("id")

		if role != auth.Doctor {
			c.JSON(403, gin.H{"error": "only doctors allowed"})
			return
		}

		err := repo.AddOverride(o)
		if err != nil {
			err := err.(endpointerrors.EndpointError)
			c.JSON(err.StatusCode, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"result": "override added"})
	}
}

func AddPermanentChange(repo WorkingHoursRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var payload struct {
			EffectiveFrom time.Time   `json:"effective_from"`
			Hour          WorkingHour `json:"hour"`
		}

		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		roleAny, _ := c.Get("role")
		role := roleAny.(byte)
		doctorID := c.GetInt("id")

		if role != auth.Doctor {
			c.JSON(403, gin.H{"error": "only doctors allowed"})
			return
		}

		err := repo.AddPermanentChange(doctorID, payload.EffectiveFrom, payload.Hour)
		if err != nil {
			err := err.(endpointerrors.EndpointError)
			c.JSON(err.StatusCode, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"result": "permanent change scheduled"})
	}
}
