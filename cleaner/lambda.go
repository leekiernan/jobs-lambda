package main

import (
	"context"
	"os"

	"github.com/aws/aws-lambda-go/lambda"

	"database/sql"

	_ "github.com/lib/pq"

	"fmt"
)

type Pagespeed struct {
	LighthouseResult struct {
		RequestedUrl      string `json:"requestedUrl"`
		LighthouseVersion string `json:"lighthouseVersion"`
		Categories        struct {
			Performance struct {
				Id    string  `json:"id"`
				Title string  `json:"title"`
				Score float32 `json:"score"`
			} `json:"performance"`
		} `json:"categories"`
	} `json:"lighthouseResult"`
}

type Job struct {
	id       int
	pageID   int
	attempts int
	key      string
	priority int
	strategy string
	url      string
}

// Result -
type Result struct {
	id       int
	pageID   int
	attempts int
	data     Pagespeed
	strategy string
}

var host = os.Getenv("RDS_HOSTNAME")
var port = os.Getenv("RDS_PORT")
var name = os.Getenv("RDS_DB_NAME")
var user = os.Getenv("RDS_USERNAME")
var pass = os.Getenv("RDS_PASSWORD")

// ConnectionParams A comment
const ConnectionParams = ""

// DBCon database connection.
var (
	DBCon *sql.DB
)

// HandleRequest for Lambda
func HandleRequest(ctx context.Context) (string, error) {
	db, err := sql.Open("postgres", ConnectionParams)
	DBCon = db

	if err != nil {
		return fmt.Sprintf("Err %s", err), err
	}

	defer DBCon.Close()

	var (
		jobID int
	)

	rows, err := DBCon.Query(`SELECT j.id from Jobs j WHERE j.running = true`)

	if err != nil {
		return fmt.Sprintf("Checker Query Error.", err), nil
	}

	defer rows.Close()
	for rows.Next() {
		err := rows.Scan(&jobID)


		if err != nil {
			return fmt.Sprintf("Rows error."), nil
		}

		DBCon.QueryRow("UPDATE Jobs SET running = false WHERE id = $1", jobID)
	}

	err = rows.Err()
	if err != nil {
		return fmt.Sprintf("Error rows", err), nil
	}

	return fmt.Sprintf("Jobs are done."), nil
}

func main() {
	lambda.Start(HandleRequest)
}
