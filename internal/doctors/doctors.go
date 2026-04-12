package doctors

import (
	"net/http"

	endpointerrors "github.com/Phantomvv1/gp_software_dev_project/internal/endpoint_errors"
	"github.com/gin-gonic/gin"
)

type Doctor struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	Email          string `json:"email"`
	Address        string `json:"address"`
	WorkingHoursID *int   `json:"working_hours_id,omitempty"`
}

func RegisterDoctor(repository doctorsRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		doctor := Doctor{}
		if err := c.ShouldBindJSON(&doctor); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		createdDoctor, err := repository.Register(doctor)
		if err != nil {
			err := err.(endpointerrors.EndpointError)
			c.JSON(err.StatusCode, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"result": createdDoctor})
	}
}

func GetAllDoctors(repo doctorsRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit := c.Query("limit")

		doctors, err := repo.GetAllDoctors(limit)
		if err != nil {
			err := err.(endpointerrors.EndpointError)
			c.JSON(err.StatusCode, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"result": doctors})
	}
}

func GetDoctorById(repository doctorsRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		doctor, err := repository.GetDoctorById(id)
		if err != nil {
			err := err.(endpointerrors.EndpointError)
			c.JSON(err.StatusCode, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"result": doctor})
	}
}

func UpdateDoctor(repository doctorsRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		doctor := Doctor{}
		if err := c.ShouldBindJSON(&doctor); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		updatedDoctor, err := repository.UpdateDoctor(id, doctor)
		if err != nil {
			err := err.(endpointerrors.EndpointError)
			c.JSON(err.StatusCode, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"result": updatedDoctor})
	}
}

func DeleteDoctor(repository doctorsRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		err := repository.DeleteDoctor(id)
		if err != nil {
			err := err.(endpointerrors.EndpointError)
			c.JSON(err.StatusCode, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"result": "doctor deleted successfully"})
	}
}
