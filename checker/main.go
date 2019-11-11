package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

var host = os.Getenv("RDS_HOSTNAME")
var port = os.Getenv("RDS_PORT")
var name = os.Getenv("RDS_DB_NAME")
var user = os.Getenv("RDS_USERNAME")
var pass = os.Getenv("RDS_PASSWORD")

// ConnectionParams A comment
const ConnectionParams = fmt.Sprintf("host=%v user=%v password=%v dbname=%v sslmode=disable", host, user, pass, name)

// DBCon database connection.
var (
	DBCon *sql.DB
)

func main() {
	DBCon, err := sql.Open("postgres", ConnectionParams)

	if err != nil {
		fmt.Println("Err %s", err)
		return
		// log.Fatal(err)
	}

	defer DBCon.Close()
	fmt.Println("Running checks.")
	var (
		pid int
		url string
		// title string
	)

	// Alternative; use sql function/trigger to alter updated_at on page row - Allowing for simpler lookup; benchmark this query?
	//	AND ((SELECT a.created_at FROM Audits a WHERE a.page_id = p.id ORDER BY created_at DESC LIMIT 1) <= now() - INTERVAL '1 DAYS' OR p.created_at <= now() - INTERVAL '1 DAYS')
	rows, err := DBCon.Query(`
		SELECT p.id, p.url FROM Pages p
		WHERE p.id NOT IN (SELECT page_id FROM Jobs)
		AND p.updated_at <= now() - INTERVAL '1 DAYS'
	`)

	if err != nil {
		log.Println("Checker Query Error.")
		return
		// log.Fatal(err)
	}

	defer rows.Close()
	for rows.Next() {
		err := rows.Scan(&pid, &url)
		if err != nil {
			log.Println("Rows error.")
			return
			// log.Fatal(err)
		}

		// fmt.Println(fmt.Sprintf("====> INSERT INTO Jobs (page_id, strategy, created_at, updated_at) VALUES (%d, 'desktop', now(), now())", pid))
		// fmt.Println(fmt.Sprintf("====> INSERT INTO Jobs (page_id, strategy, created_at, updated_at) VALUES (%d, 'mobile', now(), now())", pid))
		DBCon.QueryRow("INSERT INTO Jobs (page_id, strategy, created_at, updated_at) VALUES ($1, $2, now(), now())", pid, "desktop")
		DBCon.QueryRow("INSERT INTO Jobs (page_id, strategy, created_at, updated_at) VALUES ($1, $2, now(), now())", pid, "mobile")
	}

	err = rows.Err()
	if err != nil {
		fmt.Println("Error rows", err)
		return
		// log.Fatal(err)
	}

	fmt.Println("Checks are done.")
}
