package main

import (
	"net/http"

	"github.com/Phantomvv1/gp_software_dev_project/internal/doctors"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.Any("", func(c *gin.Context) { c.JSON(http.StatusOK, nil) })

	doc := r.Group("/doctors")
	doctorRepository := doctors.ProdRepository{}

	doc.GET("/:id", doctors.GetDoctorById(doctorRepository))
	doc.GET("", doctors.GetAllDoctors(doctorRepository))
	doc.POST("", doctors.RegisterDoctor(doctorRepository))
	doc.PUT("/:id", doctors.UpdateDoctor(doctorRepository))
	doc.DELETE("/:id", doctors.DeleteDoctor(doctorRepository))

	r.Run(":42069")
}
