package main
import "log"
import "encoding/json"
import (
  _ "github.com/lib/pq"
  "database/sql"
)
import "os"
import "strings"
import "fmt"
import "net/http"

func queryDatabase(query string) map[string] interface{} {
	var rowsSlice []map[string]string
  result := make(map[string] interface {})
  pgurl := fmt.Sprintf("%s/%s?application_name=backend", os.Getenv("OPENSHIFT_POSTGRESQL_DB_URL"), os.Getenv("PGDATABASE"))
  db, err := sql.Open("postgres", pgurl)
	if err != nil {
		log.Print(err)
	}
	defer db.Close()

	// Execute the query
	rows, err := db.Query(query)
	if err != nil {
		panic(err.Error()) // proper error handling instead of panic in your app
	}

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		panic(err.Error()) // proper error handling instead of panic in your app
	}

	// Make a slice for the values
	values := make([]sql.RawBytes, len(columns))

	// rows.Scan wants '[]interface{}' as an argument, so we must copy the
	// references into such a slice
	// See http://code.google.com/p/go-wiki/wiki/InterfaceSlice for details
	scanArgs := make([]interface{}, len(values))
	for i := range values {
		scanArgs[i] = &values[i]
	}

	// Fetch rows
	for rows.Next() {
		// get RawBytes from data
		err = rows.Scan(scanArgs...)
		if err != nil {
			panic(err.Error()) // proper error handling instead of panic in your app
		}

		// Now do something with the data.
		// Here we just print each column as a string.
		var value string
		valuesMap := make(map[string]string)
		for i, col := range values {
			// Here we can check if the value is nil (NULL value)
			if col == nil {
				value = ""
			} else {
				value = string(col)
			}
			valuesMap[columns[i]] = value
		}
		rowsSlice = append(rowsSlice, valuesMap)
	}
	result["result"] = rowsSlice
	result["count"] = len(rowsSlice)
	log.Print("[db] found: ", result["count"])
	return result
}

func handler(w http.ResponseWriter, r *http.Request) {
  enc := json.NewEncoder(w)
  q := r.URL.Query()
  log.Print("[http] ", r.URL)
  
  if r.URL.Path == "/tasks/"{
    if len(q) == 0 {
      enc.Encode(queryDatabase("SELECT id, status FROM tasks"))
    } else {
      if val, ok := q["id"]; ok {
        decoder := json.NewDecoder(r.Body)
        var params map[string] []string
        //var fields string = "*"
        err := decoder.Decode(&params)
        if err == nil {
          fields := strings.Join(params["fields"], ",")
          query := fmt.Sprintf("SELECT %s FROM tasks WHERE id = %s", fields, val[0])
          enc.Encode(queryDatabase(query))
        } else {
          w.WriteHeader(403)
        }
      }
    }
  } 
}

func main() {
  http.HandleFunc("/", handler)
  addr := fmt.Sprintf("%s:%s", os.Getenv("OPENSHIFT_GO_IP"), os.Getenv("OPENSHIFT_GO_PORT"))
  http.ListenAndServe(addr, nil)
}
