package viewer

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

type Config struct {
	DBPath string
	Port   string
}

var (
	lastModified int64
)

type ColumnInfo struct {
	CID          int         `json:"cid"`
	Name         string      `json:"name"`
	Type         string      `json:"type"`
	NotNull      int         `json:"notnull"`
	DefaultValue interface{} `json:"default_value"`
	PK           int         `json:"pk"`
}

type DBStatus struct {
	Exists   bool  `json:"exists"`
	Size     int64 `json:"size"`
	Modified int64 `json:"modified"`
	Changed  bool  `json:"changed"`
}

type QueryRequest struct {
	Query string `json:"query"`
}

type QueryResponse struct {
	Results  []map[string]interface{} `json:"results,omitempty"`
	Message  string                   `json:"message,omitempty"`
	RowCount int                      `json:"rowCount"`
	Error    string                   `json:"error,omitempty"`
}

func StartServer(config Config) error {
	log.Printf("Starting DB Viewer on port %s\n", config.Port)

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/schema", func(w http.ResponseWriter, r *http.Request) {
		schemaHandler(w, config.DBPath)
	})
	http.HandleFunc("/data/", func(w http.ResponseWriter, r *http.Request) {
		dataHandler(w, r, config.DBPath)
	})
	http.HandleFunc("/db_status", func(w http.ResponseWriter, r *http.Request) {
		dbStatusHandler(w, config.DBPath)
	})
	http.HandleFunc("/query", func(w http.ResponseWriter, r *http.Request) {
		queryHandler(w, r, config.DBPath)
	})

	return http.ListenAndServe(":"+config.Port, nil)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "templates/index.html")
}

func schemaHandler(w http.ResponseWriter, dbPath string) {
	w.Header().Set("Content-Type", "application/json")

	if !fileExists(dbPath) {
		writeJSONError(w, "Database file not found")
		return
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Printf("Error opening database: %v", err)
		writeJSONError(w, fmt.Sprintf("Error opening database: %v", err))
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table';")
	if err != nil {
		log.Printf("Error querying tables: %v", err)
		writeJSONError(w, fmt.Sprintf("Error querying tables: %v", err))
		return
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			log.Printf("Error scanning table name: %v", err)
			continue
		}
		tables = append(tables, tableName)
	}

	schema := make(map[string][]ColumnInfo)
	for _, table := range tables {
		tableInfo, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s);", table))
		if err != nil {
			log.Printf("Error getting schema for table %s: %v", table, err)
			continue
		}

		var columns []ColumnInfo
		for tableInfo.Next() {
			var col ColumnInfo
			var defaultValue interface{}
			if err := tableInfo.Scan(&col.CID, &col.Name, &col.Type, &col.NotNull, &defaultValue, &col.PK); err != nil {
				log.Printf("Error scanning column info: %v", err)
				continue
			}
			col.DefaultValue = defaultValue
			columns = append(columns, col)
		}
		tableInfo.Close()
		schema[table] = columns
	}

	json.NewEncoder(w).Encode(schema)
}

func dataHandler(w http.ResponseWriter, r *http.Request, dbPath string) {
	w.Header().Set("Content-Type", "application/json")

	if !fileExists(dbPath) {
		writeJSONError(w, "Database file not found")
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		writeJSONError(w, "Invalid table path")
		return
	}
	table := parts[2]

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Printf("Error opening database: %v", err)
		writeJSONError(w, fmt.Sprintf("Error opening database: %v", err))
		return
	}
	defer db.Close()

	colRows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s);", table))
	if err != nil {
		log.Printf("Error getting columns for table %s: %v", table, err)
		writeJSONError(w, fmt.Sprintf("Error getting columns for table %s: %v", table, err))
		return
	}
	defer colRows.Close()

	var columns []string
	for colRows.Next() {
		var cid, notNull, pk int
		var name, colType string
		var defaultValue interface{}
		if err := colRows.Scan(&cid, &name, &colType, &notNull, &defaultValue, &pk); err != nil {
			log.Printf("Error scanning column info: %v", err)
			continue
		}
		columns = append(columns, name)
	}

	dataRows, err := db.Query(fmt.Sprintf("SELECT * FROM %s LIMIT 1000;", table))
	if err != nil {
		log.Printf("Error querying data from table %s: %v", table, err)
		writeJSONError(w, fmt.Sprintf("Error querying data from table %s: %v", table, err))
		return
	}
	defer dataRows.Close()

	columnValues := make([]interface{}, len(columns))
	columnValuePtrs := make([]interface{}, len(columns))
	for i := range columnValues {
		columnValuePtrs[i] = &columnValues[i]
	}

	var data []map[string]interface{}
	for dataRows.Next() {
		if err := dataRows.Scan(columnValuePtrs...); err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}

		rowData := make(map[string]interface{})
		for i, colName := range columns {
			val := columnValues[i]
			switch v := val.(type) {
			case []uint8:
				rowData[colName] = string(v)
			default:
				rowData[colName] = v
			}
		}
		data = append(data, rowData)
	}

	json.NewEncoder(w).Encode(data)
}

func dbStatusHandler(w http.ResponseWriter, dbPath string) {
	w.Header().Set("Content-Type", "application/json")

	status := DBStatus{
		Exists:   false,
		Size:     0,
		Modified: 0,
		Changed:  false,
	}

	if !fileExists(dbPath) {
		json.NewEncoder(w).Encode(status)
		return
	}

	fileInfo, err := os.Stat(dbPath)
	if err != nil {
		log.Printf("Error getting file info: %v", err)
		json.NewEncoder(w).Encode(status)
		return
	}

	currentModified := fileInfo.ModTime().Unix()
	status.Exists = true
	status.Size = fileInfo.Size()
	status.Modified = currentModified
	status.Changed = currentModified > lastModified

	if status.Changed {
		lastModified = currentModified
	}

	json.NewEncoder(w).Encode(status)
}

func queryHandler(w http.ResponseWriter, r *http.Request, dbPath string) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !fileExists(dbPath) {
		writeJSONError(w, "Database file not found")
		return
	}

	var queryReq QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&queryReq); err != nil {
		writeJSONError(w, fmt.Sprintf("Error parsing request: %v", err))
		return
	}

	if queryReq.Query == "" {
		writeJSONError(w, "No query provided")
		return
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Printf("Error opening database: %v", err)
		writeJSONError(w, fmt.Sprintf("Error opening database: %v", err))
		return
	}
	defer db.Close()

	queryUppercase := strings.TrimSpace(strings.ToUpper(queryReq.Query))
	if strings.HasPrefix(queryUppercase, "SELECT") {
		rows, err := db.Query(queryReq.Query)
		if err != nil {
			log.Printf("SQLite error: %v", err)
			writeJSONError(w, fmt.Sprintf("SQLite error: %v", err))
			return
		}
		defer rows.Close()

		columns, err := rows.Columns()
		if err != nil {
			log.Printf("Error getting columns: %v", err)
			writeJSONError(w, fmt.Sprintf("Error getting columns: %v", err))
			return
		}

		columnValues := make([]interface{}, len(columns))
		columnValuePtrs := make([]interface{}, len(columns))
		for i := range columnValues {
			columnValuePtrs[i] = &columnValues[i]
		}

		var results []map[string]interface{}
		for rows.Next() {
			if err := rows.Scan(columnValuePtrs...); err != nil {
				log.Printf("Error scanning row: %v", err)
				continue
			}

			rowData := make(map[string]interface{})
			for i, colName := range columns {
				val := columnValues[i]
				switch v := val.(type) {
				case []uint8:
					rowData[colName] = string(v)
				default:
					rowData[colName] = v
				}
			}
			results = append(results, rowData)
		}

		response := QueryResponse{
			Results:  results,
			RowCount: len(results),
		}
		json.NewEncoder(w).Encode(response)
	} else {
		result, err := db.Exec(queryReq.Query)
		if err != nil {
			log.Printf("SQLite error: %v", err)
			writeJSONError(w, fmt.Sprintf("SQLite error: %v", err))
			return
		}

		rowCount, _ := result.RowsAffected()
		response := QueryResponse{
			Message:  "Query executed successfully",
			RowCount: int(rowCount),
		}
		json.NewEncoder(w).Encode(response)
	}
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func writeJSONError(w http.ResponseWriter, errMsg string) {
	w.WriteHeader(http.StatusOK)
	resp := map[string]string{"error": errMsg}
	json.NewEncoder(w).Encode(resp)
}
