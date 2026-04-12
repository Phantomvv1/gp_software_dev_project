package auth

import (
	"bytes"
	"context"
	"crypto/sha512"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/jackc/pgx/v5"
)

const (
	Doctor = iota + 1
	Patient
)

var Domain = ""
var Secure = false

func GenerateJWT(id string, accountType byte, email string) (string, error) {
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

func ValidateJWT(tokenString string) (string, byte, string, error) {
	claims := &jwt.MapClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
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

func CreateAuthTable(conn *pgx.Conn) error {
	_, err := conn.Exec(context.Background(), "create table if not exists authentication (id uuid primary key, full_name text, "+
		"email text, password text, type int check (type in (1, 2)), created_at timestamp default current_timestamp, updated_at timestamp default current_timestamp)")
	return err
}

func CreateRefreshTokenTable(conn *pgx.Conn) error {
	_, err := conn.Exec(context.Background(), "create table if not exists r_tokens(token uuid primary key default gen_random_uuid(), user_id uuid references authentication(id) on delete cascade, "+
		"expiration timestamp default current_timestamp + '5 days'::interval, valid bool)")
	return err
}

func SignUp(c *gin.Context) {
	acc := AlpacaAccount{}
	if err := c.ShouldBindJSON(&acc); err != nil {
		ErrorExit(c, http.StatusInternalServerError, "couldn't parse the body of the request correctly", err)
		return
	}

	if acc.Password == "" {
		ErrorExit(c, http.StatusBadRequest, "password required for creating an account", nil)
		return
	}

	password := acc.Password
	acc.Password = ""

	acc.Agreements = make([]map[string]string, 1)
	acc.Agreements[0] = make(map[string]string)
	acc.Agreements[0]["agreement"] = "customer_agreement"
	acc.Agreements[0]["signed_at"] = time.Now().UTC().Format(time.RFC3339)
	acc.Agreements[0]["ip_address"] = c.ClientIP()

	req, err := json.Marshal(acc)
	if err != nil {
		ErrorExit(c, http.StatusInternalServerError, "couldn't recreate the request", err)
		return
	}

	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Databse connection failed"})
		return
	}
	defer conn.Close(context.Background())

	if err = CreateAuthTable(conn); err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating a table for authentication"})
		return
	}

	errs := map[int]string{
		400: "The post body is not well formed",
		409: "There is already an existing account registered with the same email address",
		422: "One of the input values is not a valid value",
	}

	headers := BasicAuth()

	reader := bytes.NewReader(req)

	body, err := SendRequest[AlpacaAccount](http.MethodPost, BaseURL+Accounts, reader, errs, headers)
	if err != nil {
		RequestExit(c, body, err, "unable to make an account for the user")
		return
	}

	id := body.ID
	name := body.Identity.GivenName + " " + body.Identity.FamilyName
	email := body.Contact.Email

	hashedPassword := SHA512(password)
	_, err = conn.Exec(context.Background(), "insert into authentication (id, full_name, email, password, type) values ($1, $2, $3, $4, $5)",
		id, name, email, hashedPassword, User)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error inserting the information into the database."})
		return
	}

	c.JSON(http.StatusOK, body)
}

func LogIn(c *gin.Context) {
	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection failed"})
		return
	}

	if err = CreateAuthTable(conn); err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	var information map[string]string
	json.NewDecoder(c.Request.Body).Decode(&information) //email, password

	var passwordCheck, email string
	var accoutType byte
	var id string
	err = conn.QueryRow(context.Background(), "select id, password, type, email from authentication a where a.email = $1;", information["email"]).Scan(
		&id, &passwordCheck, &accoutType, &email)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "There isn't anybody registered with this email!"})
			return
		} else {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error while trying to log in"})
			return
		}
	}

	if SHA512(information["password"]) != passwordCheck {
		log.Println("Wrong password")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Error wrong password"})
		return
	}

	jwtToken, err := GenerateJWT(id, accoutType, email)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error while generating your token"})
		return
	}

	refreshToken := ""
	err = conn.QueryRow(context.Background(), "insert into r_tokens (user_id, valid) values ($1, $2) returning token", id, true).Scan(&refreshToken)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to generate a refresh token"})
		return
	}

	c.SetCookie("refresh", refreshToken, int((5 * 24 * time.Hour).Seconds()), "/", Domain, Secure, true)
	c.JSON(http.StatusOK, gin.H{"token": jwtToken})
}

// From local DB
func GetAllUsers(c *gin.Context) {
	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to connect to the database"})
		return
	}
	defer conn.Close(context.Background())

	rows, err := conn.Query(context.Background(), "select id, full_name, email, type, created_at, updated_at from authentication")
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error couldn't get information from the database"})
		return
	}

	var profiles []Profile
	for rows.Next() {
		profile := Profile{}
		err = rows.Scan(&profile.ID, &profile.Name, &profile.Email, &profile.Type, &profile.CreatedAt, &profile.UpdatedAt)
		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error working with the data from the database"})
			return
		}

		profiles = append(profiles, profile)
	}

	if rows.Err() != nil {
		log.Println(rows.Err())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error working with the data from the database"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"users": profiles})
}

// This endpoint makes an external API call,
// only use it if you want more information about the user
func GetAllUsersAlpaca(c *gin.Context) {
	headers := BasicAuth()

	body, err := SendRequest[any](http.MethodGet, BaseURL+Accounts, nil, nil, headers)
	if err != nil {
		RequestExit(c, body, err, "unable to get all users")
		return
	}

	c.JSON(http.StatusOK, body)
}

func InvalidateRefreshTokens(conn *pgx.Conn, userID string) error {
	_, err := conn.Exec(context.Background(), "update r_tokens set valid = false where user_id = $1 and valid = true", userID)
	return err
}

func Refresh(c *gin.Context) {
	refresh, err := c.Cookie("refresh")
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to get the refresh token"})
		return
	}

	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to connect to the database"})
		return
	}
	defer conn.Close(context.Background())

	var (
		id          string
		email       string
		accountType int
		valid       bool
		expired     bool
	)

	err = conn.QueryRow(
		context.Background(),
		`
	SELECT
	    a.id,
	    a.email,
	    a.type,
	    r.valid,
	    current_timestamp > r.expiration AS expired
	FROM r_tokens r
	JOIN authentication a
	  ON a.id = r.user_id
	WHERE r.token = $1
	`,
		refresh,
	).Scan(&id, &email, &accountType, &valid, &expired)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			ErrorExit(c, http.StatusUnauthorized, "invalid refresh token", err)
			return
		}

		ErrorExit(c, http.StatusInternalServerError, "couldn't get information from the database", err)
		return
	}

	if !valid {
		err = InvalidateRefreshTokens(conn, id)
		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to invalidate the token"})
			return
		}

		c.JSON(http.StatusUnauthorized, gin.H{"error": "Error refresh token reused"})
		return
	}

	if expired {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Error expried refresh token"})
		return
	}

	_, err = conn.Exec(context.Background(), "update r_tokens set valid = false where user_id = $1", id)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to invalidate the refresh token"})
		return
	}

	newRefresh := ""
	err = conn.QueryRow(context.Background(), "insert into r_tokens (user_id, valid) values ($1, $2) returning token", id, true).Scan(&newRefresh)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to create a new refresh token"})
		return
	}

	c.SetCookie("refresh", newRefresh, int((5 * 24 * time.Hour).Seconds()), "/", Domain, Secure, true)

	token, err := GenerateJWT(id, byte(accountType), email)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to generate a new token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

// From the local DB
func GetUser(c *gin.Context) {
	id := c.GetString("id")
	acc, _ := c.Get("accountType")
	accountType := acc.(byte)

	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to connect to the database"})
		return
	}
	defer conn.Close(context.Background())

	user := Profile{}
	user.ID = id
	user.Type = accountType
	err = conn.QueryRow(context.Background(), "select full_name, email, created_at, updated_at from authentication where id = $1", user.ID).
		Scan(&user.Name, &user.Email, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to get the user from the database"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

func UpdateUser(c *gin.Context) {
	// name &| email
	id := c.GetString("id")

	name := c.GetString("name")
	email := c.GetString("json_email")
	if name == "" && email == "" {
		ErrorExit(c, http.StatusBadRequest, "no new information given", nil)
	}

	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		ErrorExit(c, http.StatusInternalServerError, "couldn't connect to the database", err)
		return
	}
	defer conn.Close(context.Background())

	if name != "" && email != "" {
		_, err = conn.Exec(context.Background(), "update authentication set full_name = $1, email = $2, updated_at = current_timestamp where id = $3", name, email, id)
		if err != nil {
			ErrorExit(c, http.StatusInternalServerError, "unable to update the person in the database", err)
			return
		}
	} else if name != "" {
		_, err = conn.Exec(context.Background(), "update authentication set full_name = $1, updated_at = current_timestamp where id = $2", name, id)
		if err != nil {
			ErrorExit(c, http.StatusInternalServerError, "unable to update the person in the database", err)
			return
		}
	} else {
		_, err = conn.Exec(context.Background(), "update authentication set email = $1, updated_at = current_timestamp where id = $2", email, id)
		if err != nil {
			ErrorExit(c, http.StatusInternalServerError, "unable to update the person in the database", err)
			return
		}
	}
}

// This endpoint makes an external API call,
// only use it if you want to update more information about the user
func UpdateUserAlpaca(c *gin.Context) {
	id := c.GetString("id")

	headers := BasicAuth()

	errs := map[int]string{
		400: "The post body is not well formed",
		422: "The response body contains an atribute that is not permited to be updated or you are atempting to set an invalid value",
	}

	body, err := SendRequest[any](http.MethodPatch, BaseURL+Accounts+id, c.Request.Body, errs, headers)
	if err != nil {
		RequestExit(c, body, err, "unable to update the user")
		return
	}

	c.JSON(http.StatusOK, body)
}

func DeleteUser(c *gin.Context) {
	id := c.GetString("id")

	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		ErrorExit(c, http.StatusInternalServerError, "unable to connect to the database", err)
		return
	}
	defer conn.Close(context.Background())

	headers := BasicAuth()

	errs := map[int]string{
		404: "Account not found",
	}

	body, err := SendRequest[any](http.MethodPost, BaseURL+Accounts+id+"/actions/close", nil, errs, headers)
	if err != nil {
		RequestExit(c, body, err, "unable to delete the account of the user")
		return
	}

	_, err = conn.Exec(context.Background(), "delete from authentication where id = $1", id)
	if err != nil {
		ErrorExit(c, http.StatusInternalServerError, "deleting the person from the database", err)
		return
	}

	c.JSON(http.StatusOK, body)
}

// All the profile information
func GetUserAlpaca(c *gin.Context) {
	id := c.GetString("id")

	headers := BasicAuth()

	body, err := SendRequest[any](http.MethodGet, BaseURL+Accounts+id, nil, nil, headers)
	if err != nil {
		RequestExit(c, body, err, "unable to get the account of the user")
		return
	}

	c.JSON(http.StatusOK, body)
}

func GetAccountTradingDetails(c *gin.Context) {
	id := c.GetString("id")

	headers := BasicAuth()

	body, err := SendRequest[TradingDetails](http.MethodGet, BaseURL+Trading+id+"/account", nil, nil, headers)
	if err != nil {
		RequestExit(c, body, err, "unable to get the trading details of the account")
		return
	}

	c.JSON(http.StatusOK, body)
}
