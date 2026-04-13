package routes

import (
	"net/http"

	"github.com/Phantomvv1/gp_software_dev_project/internal/auth"
	"github.com/Phantomvv1/gp_software_dev_project/internal/doctors"
	"github.com/Phantomvv1/gp_software_dev_project/internal/middleware"
	"github.com/Phantomvv1/gp_software_dev_project/internal/patients"
	"github.com/gin-gonic/gin"
)

func GetRoutes() *gin.Engine {
	authRepo := auth.ProdRepository{}
	doctorRepository := doctors.ProdRepository{}
	patientsRepo := patients.ProdRepository{}

	r := gin.Default()

	r.Any("", middleware.APIKeyAuthMiddleware, func(c *gin.Context) { c.JSON(http.StatusOK, nil) })
	r.POST("/doctors", middleware.APIKeyAuthMiddleware, doctors.RegisterDoctor(doctorRepository))
	r.POST("/patients", middleware.APIKeyAuthMiddleware, patients.RegisterPatient(patientsRepo))
	r.POST("/login", middleware.APIKeyAuthMiddleware, auth.Login(authRepo))

	protected := r.Group("")
	protected.Use(middleware.AuthMiddleware)

	protected.GET("/me", func(c *gin.Context) {
		roleAny, _ := c.Get("role")
		role := roleAny.(byte)
		c.JSON(200, gin.H{
			"id":    c.GetInt("user_id"),
			"email": c.GetString("email"),
			"role":  role,
		})
	})

	addDoctorRoutes(protected, doctorRepository)
	addPatientRoutes(protected, patientsRepo)

	return r
}

func addDoctorRoutes(r *gin.RouterGroup, repo doctors.DoctorsRepository) {
	doc := r.Group("/doctors")

	doc.GET("/:id", doctors.GetDoctorById(repo))
	doc.GET("", doctors.GetAllDoctors(repo))
	doc.PUT("/:id", doctors.UpdateDoctor(repo))
	doc.DELETE("/:id", doctors.DeleteDoctor(repo))
}

func addPatientRoutes(r *gin.RouterGroup, repo patients.PatientsRepository) {
	pats := r.Group("/patients")

	pats.GET("", patients.GetAllPatients(repo))
	pats.GET("/:id", patients.GetPatientById(repo))
	pats.PUT("/:id", patients.UpdatePatient(repo))
	pats.DELETE("/:id", patients.DeletePatient(repo))
}
