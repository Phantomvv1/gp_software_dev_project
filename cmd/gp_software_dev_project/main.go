package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.Any("", func(c *gin.Context) { c.JSON(http.StatusOK, nil) })
}
