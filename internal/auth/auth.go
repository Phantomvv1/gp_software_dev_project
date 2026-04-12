package auth

import (
	"crypto/sha512"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	endpointerrors "github.com/Phantomvv1/gp_software_dev_project/internal/endpoint_errors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

const (
	Doctor = iota + 1
	Patient
)

var Domain = ""
var Secure = false

var (
	invalidCredentialsError = errors.New("Error: invalid email or password")
	dbConnectionError       = errors.New("Error: unable to connect to database")
	tokenGenerationError    = errors.New("Error: unable to generate token")
	ParsingTokenError       = errors.New("Error: invalid token")
)

func GenerateJWT(id int, accountType byte, email string) (string, error) {
	claims := jwt.MapClaims{
		"id":         id,
		"type":       accountType,
		"email":      email,
		"expiration": time.Now().Add(time.Minute * 15).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	jwtKey := os.Getenv("JWT_KEY")
	return token.SignedString([]byte(jwtKey))
}

// id, accountType, email, err
func ValidateJWT(tokenString string) (string, byte, string, error) {
	claims := &jwt.MapClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.ErrUnsupported
		}

		return []byte(os.Getenv("JWT_KEY")), nil
	})

	if err != nil || !token.Valid {
		return "", 0, "", err
	}

	expiration, ok := (*claims)["expiration"].(float64)
	if !ok {
		return "", 0, "", errors.New("Error parsing the expiration date of the token")
	}

	if int64(expiration) < time.Now().Unix() {
		return "", 0, "", errors.New("Error token has expired")
	}

	id, ok := (*claims)["id"].(string)
	if !ok {
		return "", 0, "", errors.New("Error parsing the id")
	}

	accountType, ok := (*claims)["type"].(float64)
	if !ok {
		return "", 0, "", errors.New("Error parsing the account")
	}

	email, ok := (*claims)["email"].(string)
	if !ok {
		return "", 0, "", errors.New("Error parsing the email")
	}

	return id, byte(accountType), email, nil
}

func SHA512(text string) string {
	algorithm := sha512.New()
	algorithm.Write([]byte(text))
	result := algorithm.Sum(nil)
	return fmt.Sprintf("%x", result)
}

func Login(repo authRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}

		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		id, email, role, err := repo.Login(body.Email, body.Password)
		if err != nil {
			err := err.(endpointerrors.EndpointError)
			c.JSON(err.StatusCode, gin.H{"error": err.Error()})
			return
		}

		token, err := GenerateJWT(id, role, email)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": tokenGenerationError.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"token": token})
	}
}
