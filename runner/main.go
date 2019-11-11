package main

import (
	"database/sql"
	"os"

	_ "github.com/lib/pq"

	"fmt"
	"log"

	// "reflect"

	"encoding/json"
	"io/ioutil"
	"net/http"
)

// Pagespeed datastruct from google.
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

// Job -
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
const ConnectionParams = fmt.Sprintf("host=%v user=%v password=%v dbname=%v sslmode=disable", host, user, pass, name)

// DBCon database connection.
var (
	DBCon *sql.DB
)

func main() {
	// db, err := sql.Open("postgres", CONNECTION_PARAMS)
	db, err := sql.Open("postgres", CONNECTION_PARAMS)
	DBCon = db

	if err != nil {
		fmt.Println("Err %s", err)
		log.Fatal(err)
	}

	defer DBCon.Close()
	fmt.Println("Running updates.")
	var (
		attempts int
		id       int
		apiKey   string
		pageID   int
		priority int
		strategy string
		url      string
	)

	rows, err := DBCon.Query(`SELECT j.id, j.page_id, j.attempts, j.priority, j.strategy, p.url, c.pagespeed_api_key
		FROM Jobs j
		INNER JOIN Pages p ON p.id = j.page_id
		INNER JOIN Clients c on c.id = p.client_id
		WHERE j.running = false
		ORDER BY j.priority DESC`)

	if err != nil {
		log.Println("Runner Query Error.")
		log.Fatal(err)
	}

	var countRows int
	err = DBCon.QueryRow(`SELECT COUNT(*) FROM Jobs WHERE running = false`).Scan(&countRows)

	if err != nil {
		log.Println("Runner Query Error @ getting count.")
		log.Fatal(err)
	} else {
		log.Println("The count is ", countRows)
	}

	defer rows.Close()

	jobs := make(chan Job, countRows)
	results := make(chan Result, countRows)

	go fetch(jobs, results)
	go fetch(jobs, results)
	go fetch(jobs, results)
	go fetch(jobs, results)

	for rows.Next() {
		err := rows.Scan(&id, &pageID, &attempts, &priority, &strategy, &url, &apiKey)
		fmt.Println("Auditing", url)
		if err != nil {
			log.Println("Runner Rows error.")
			log.Fatal(err)
		}

		jobs <- Job{
			id:       id,
			pageID:   pageID,
			attempts: attempts,
			key:      apiKey,
			priority: priority,
			strategy: strategy,
			url:      url}
		// url: fmt.Sprintf("https://www.googleapis.com/pagespeedonline/v5/runPagespeed?url=%s&locale=en-gb&strategy=mobile&utm_campaign=pagespeed_dashboarding&utm_source=equimedia&fields=lighthouseResult(categories(performance(description,id,manualDescription,score,title)),lighthouseVersion,requestedUrl)&key=%s", url, apiKey) }
	}

	close(jobs)

	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	// for r := range results {}
	for i := 0; i < countRows; i++ {
		var r Result = <-results
		fmt.Println("Result", r.page_id, r.strategy)

		stmt, err := DBCon.Prepare("INSERT INTO Audits (strategy,performance,lighthouse_versio,page_id,created_at,updated_at) VALUES ($1, $2, $3, $4, now(), now())")
		stmt.Exec(string(r.strategy), int(r.data.LighthouseResult.Categories.Performance.Score*100), string(r.data.LighthouseResult.LighthouseVersion), r.page_id)

		if err != nil {
			fmt.Println("Error inserting", err)
		} else {
			fmt.Println("Attempting remove ", r.id)
			stmt, _ = DBCon.Prepare("DELETE FROM Jobs WHERE id = $1")
			stmt.Exec(r.id)

			if err != nil {
				fmt.Println("Error removing job", err)
			}
		}
	}

	fmt.Println("Jobs are done.")
}

func fetch(jobs <-chan Job, results chan<- Result) {
	// DBCon, err := sql.Open("postgres", CONNECTION_PARAMS)

	// Loop and pull from the job channel.
	for job := range jobs {
		fmt.Println("===> ", fmt.Sprintf("UPDATE Jobs SET running = true WHERE id = %d", job.id))

		// if DBCon == nil {
		// 	fmt.Println("No DB connection.")
		// }

		stmt, err := DBCon.Prepare("UPDATE Jobs SET running = true WHERE id = $1")
		if err != nil {
			fmt.Println("Error updating job 1")
		}

		_, err = stmt.Exec(job.id)
		if err != nil {
			fmt.Println("Error updating job 1")
		}

		url := fmt.Sprintf("https://www.googleapis.com/pagespeedonline/v5/runPagespeed?url=%s&locale=en-gb&strategy=%s&utm_campaign=pagespeed_dashboarding&utm_source=equimedia&fields=lighthouseResult(categories(performance(description,id,manualDescription,score,title)),lighthouseVersion,requestedUrl)&key=%s", job.url, job.strategy, job.key)
		fmt.Println(url)
		res, err := http.Get(url)

		if err != nil {
			fmt.Println("ERROR fetching, ", url, err)
		}

		if res.StatusCode >= 200 && res.StatusCode < 300 {
			fmt.Println("Success...")

			defer res.Body.Close()
			data, _ := ioutil.ReadAll(res.Body)

			var ps Pagespeed
			err := json.Unmarshal(data, &ps)
			if err != nil {
				fmt.Println("Error marshalling json", err)
			}

			// Push to results channel when done.
			results <- Result{
				id:       job.id,
				pageID:   job.pageID,
				attempts: job.attempts + 1,
				strategy: job.strategy,
				data:     ps}
		} else {
			fmt.Println("Failure in API.")
		}

		stmt, err = DBCon.Prepare("UPDATE Jobs SET running = false, attempts = $1 WHERE id = $2")
		stmt.Exec((job.attempts + 1), job.id)
		if err != nil {
			fmt.Println("Error updating job 1")
		}
	}

	// 	// Set running.
	// 	// Create thread
	// 	// 	API Request to pagespeed
	// 	// 	If OK
	// 	// 		Remove Job
	// 	// 		Add Audit
	// 	// 	Else
	// 	// 		Update attempts
	// 	// 	If attempts > 5
	// 	// 		Something? Remove or raise
}
