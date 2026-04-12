package patients

import (
	endpointerrors "github.com/Phantomvv1/gp_software_dev_project/internal/endpoint_errors"
	"github.com/gin-gonic/gin"
)

type Patient struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	PhoneNumber string `json:"phone_number"`
	DoctorID    *int   `json:"doctor_id"`
	Password    string `json:"password,omitempty"`
}

func RegisterPatient(repo PatientsRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		patient := Patient{}

		if err := c.ShouldBindJSON(&patient); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		created, err := repo.Register(patient)
		if err != nil {
			err := err.(endpointerrors.EndpointError)
			c.JSON(err.StatusCode, gin.H{"error": err.Error()})
			return
		}

		c.JSON(201, gin.H{"result": created})
	}
}

func GetAllPatients(repo PatientsRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit := c.Query("limit")

		res, err := repo.GetAllPatients(limit)
		if err != nil {
			err := err.(endpointerrors.EndpointError)
			c.JSON(err.StatusCode, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"result": res})
	}
}

func GetPatientById(repo PatientsRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		res, err := repo.GetPatientById(id)
		if err != nil {
			err := err.(endpointerrors.EndpointError)
			c.JSON(err.StatusCode, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"result": res})
	}
}

func UpdatePatient(repo PatientsRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		p := Patient{}
		if err := c.ShouldBindJSON(&p); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		res, err := repo.UpdatePatient(id, p)
		if err != nil {
			err := err.(endpointerrors.EndpointError)
			c.JSON(err.StatusCode, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"result": res})
	}
}

func DeletePatient(repo PatientsRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		err := repo.DeletePatient(id)
		if err != nil {
			err := err.(endpointerrors.EndpointError)
			c.JSON(err.StatusCode, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"result": "patient deleted"})
	}
}
