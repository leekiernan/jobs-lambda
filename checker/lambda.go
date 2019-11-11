package main

import (
	"context"
	"os"

	"github.com/aws/aws-lambda-go/lambda"

	"database/sql"

	_ "github.com/lib/pq"

	"fmt"
	"log"
)

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
	log.Print("%s", ConnectionParams)
	DBCon, err := sql.Open("postgres", ConnectionParams)

	if err != nil {
		return fmt.Sprintf("Err %s", err), nil
		// log.Fatal(err)
	}

	defer DBCon.Close()
	fmt.Sprintf("Running checks.")
	var (
		pid int
		url string
		// title string
	)

	rows, err := DBCon.Query(`
		SELECT p.id, p.url FROM Pages p
		WHERE p.id NOT IN (SELECT page_id FROM Jobs)
		AND p.updated_at <= now() - INTERVAL '1 DAYS'
	`)

	if err != nil {
		return fmt.Sprintf("Checker Query Error.", err), nil
		// log.Fatal(err)
	}

	defer rows.Close()
	for rows.Next() {
		err := rows.Scan(&pid, &url)
		if err != nil {
			return fmt.Sprintf("Rows error."), nil
			// log.Fatal(err)
		}

		DBCon.QueryRow("INSERT INTO Jobs (page_id, strategy, created_at, updated_at) VALUES ($1, $2, now(), now())", pid, "desktop")
		DBCon.QueryRow("INSERT INTO Jobs (page_id, strategy, created_at, updated_at) VALUES ($1, $2, now(), now())", pid, "mobile")
	}

	err = rows.Err()
	if err != nil {
		return fmt.Sprintf("Error rows", err), nil
		// log.Fatal(err)
	}

	return fmt.Sprintf("Checks are done."), nil
}

func main() {
	lambda.Start(HandleRequest)
}
