package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/gorilla/mux"
)

var logger *log.Logger

func init() {
	// Create a log file or use os.Stdout for console logging
	logFile, err := os.OpenFile("server.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("Error opening log file: %v", err)
	}

	// Customize the logger (prefix, flags, etc.)
	logger = log.New(logFile, "EmployeeAPI: ", log.Ldate|log.Ltime|log.Lshortfile)
}

// Middleware for logging requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Printf("Request: %s %s", r.Method, r.RequestURI)
		next.ServeHTTP(w, r)
	})
}

// Employee struct
type Employee struct {
	ID       int     `json:"id"`
	Name     string  `json:"name"`
	Position string  `json:"position"`
	Salary   float64 `json:"salary"`
}

// In-memory employee database with a mutex for concurrency
var (
	employees = make(map[int]*Employee)
	mutex     sync.RWMutex
	lastID    = 0
)

// CRUD operations with concurrency safety

func CreateEmployee(w http.ResponseWriter, r *http.Request) {
	var emp Employee
	if err := json.NewDecoder(r.Body).Decode(&emp); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	mutex.Lock()
	lastID++
	emp.ID = lastID
	employees[emp.ID] = &emp
	mutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(emp)
}

func GetEmployeeByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		logger.Printf("Error: Invalid employee ID: %s", idStr)
		http.Error(w, "Invalid employee ID", http.StatusBadRequest)
		return
	}
	mutex.RLock() // Read lock to allow concurrent reads
	defer mutex.RUnlock()

	emp, ok := employees[id]

	if !ok {
		logger.Printf("Error: Employee not found (ID: %d)", id)
		http.Error(w, "Employee not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(emp)
}

func UpdateEmployee(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		logger.Printf("Error: Invalid employee ID: %s", idStr)
		http.Error(w, "Invalid employee ID", http.StatusBadRequest)
		return
	}

	if r.Header.Get("Content-Type") != "application/json" {
		logger.Printf("Error: Invalid content-type. Expected 'application/json', got '%s'", r.Header.Get("Content-Type"))
		http.Error(w, "Invalid content-type. Expected 'application/json'", http.StatusUnsupportedMediaType)
		return
	}

	var updatedEmp Employee
	if err := json.NewDecoder(r.Body).Decode(&updatedEmp); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	updatedEmp.ID = id // Ensure the ID is not changed

	mutex.Lock() // Exclusive lock to prevent concurrent modifications
	defer mutex.Unlock()

	if _, ok := employees[id]; !ok {
		logger.Printf("Error: Employee not found (ID: %d)", id)
		http.Error(w, "Employee not found", http.StatusNotFound)
		return
	}

	employees[id] = &updatedEmp
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedEmp)
}

func DeleteEmployee(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		logger.Printf("Error: Invalid employee ID: %s", idStr)
		http.Error(w, "Invalid employee ID", http.StatusBadRequest)
		return
	}
	mutex.Lock() // Exclusive lock to prevent concurrent modifications
	defer mutex.Unlock()

	if _, ok := employees[id]; !ok {
		logger.Printf("Error: Employee not found (ID: %d)", id)
		http.Error(w, "Employee not found", http.StatusNotFound)
		return
	}

	delete(employees, id)
	w.WriteHeader(http.StatusNoContent) // 204 No Content on successful delete
}

// RESTful API with pagination
func ListEmployees(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if pageSize < 1 {
		pageSize = 10
	}

	mutex.RLock()
	defer mutex.RUnlock()

	employeeList := make([]*Employee, 0, len(employees)) // Create a slice to hold employees in a specific order
	for _, emp := range employees {
		employeeList = append(employeeList, emp)
	}

	totalEmployees := len(employeeList)
	startIdx := (page - 1) * pageSize

	if startIdx >= totalEmployees {
		results := []*Employee{}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
		return
	}

	endIdx := startIdx + pageSize
	if endIdx > totalEmployees {
		endIdx = totalEmployees
	}

	results := employeeList[startIdx:endIdx]

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func main() {

	r := mux.NewRouter()

	// Define routes with specified HTTP methods
	r.HandleFunc("/employees", CreateEmployee).Methods("POST")
	r.HandleFunc("/employees/{id}", GetEmployeeByID).Methods("GET")
	r.HandleFunc("/employees/{id}", UpdateEmployee).Methods("PUT")
	r.HandleFunc("/employees/{id}", DeleteEmployee).Methods("DELETE")
	r.HandleFunc("/employees", ListEmployees).Methods("GET")

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Employee Management API")
	})

	// Use the logging middleware for all routes
	r.Use(loggingMiddleware)

	port := ":4000"
	logger.Printf("Server starting on port %s", port)
	err := http.ListenAndServe(port, r)
	if err != nil {
		logger.Fatalf("Error starting server: %v", err)
	}
}
