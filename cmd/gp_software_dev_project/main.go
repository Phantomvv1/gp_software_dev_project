package main

import (
	"github.com/Phantomvv1/gp_software_dev_project/routes"
)

func main() {
	r := routes.GetRoutes()

	r.Run(":42069")
}
