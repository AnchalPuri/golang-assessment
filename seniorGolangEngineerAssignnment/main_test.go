package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

func TestCreateEmployee(t *testing.T) {
	// Test data
	newEmp := Employee{
		Name:     "Alice Johnson",
		Position: "Software Engineer",
		Salary:   85000.0,
	}

	// Prepare request
	empJSON, _ := json.Marshal(newEmp)

	// Create a test server using your router
	router := mux.NewRouter()
	router.HandleFunc("/employees", CreateEmployee).Methods("POST")
	router.Use(loggingMiddleware) // Apply middleware to the router

	ts := httptest.NewServer(router)
	defer ts.Close()

	req, err := http.NewRequest("POST", ts.URL+"/employees", bytes.NewBuffer(empJSON))
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Content-Type", "application/json") // Set the Content-Type header

	// Create a client to send the request through the server
	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		t.Fatalf("Error sending request: %v", err)
	}
	defer resp.Body.Close()

	// Validate response
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status %v, got %v", http.StatusOK, resp.StatusCode)
	}

	var createdEmp Employee
	if err := json.NewDecoder(resp.Body).Decode(&createdEmp); err != nil {
		t.Fatal("Failed to unmarshal response:", err)
	}
	if createdEmp.Name != newEmp.Name || createdEmp.Position != newEmp.Position || createdEmp.Salary != newEmp.Salary {
		t.Errorf("Created employee data does not match: got %v, want %v", createdEmp, newEmp)
	}
}

func TestGetEmployeeByID(t *testing.T) {
	// Setup
	mutex.Lock()
	employees = make(map[int]*Employee)
	lastID = 0
	mutex.Unlock()

	newEmp := Employee{
		Name:     "John Doe",
		Position: "Manager",
		Salary:   120000.0,
	}
	CreateEmployeeHelper(t, &newEmp)

	// Create a test server using your router
	router := mux.NewRouter()
	router.HandleFunc("/employees/{id}", GetEmployeeByID).Methods("GET")
	router.Use(loggingMiddleware)

	ts := httptest.NewServer(router)
	defer ts.Close()

	// Prepare request
	req, err := http.NewRequest("GET", ts.URL+fmt.Sprintf("/employees/%d", newEmp.ID), nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a client to send the request through the server
	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		t.Fatalf("Error sending request: %v", err)
	}
	defer resp.Body.Close()

	// Validate response
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status %v, got %v", http.StatusOK, resp.StatusCode)
	}

	var fetchedEmp Employee
	if err := json.NewDecoder(resp.Body).Decode(&fetchedEmp); err != nil {
		t.Fatal("Failed to unmarshal response:", err)
	}
	if fetchedEmp != newEmp {
		t.Errorf("Fetched employee data does not match: got %v, want %v", fetchedEmp, newEmp)
	}
}

func TestUpdateEmployee(t *testing.T) {

	mutex.Lock()
	employees = make(map[int]*Employee)
	lastID = 0
	mutex.Unlock()

	newEmp := Employee{
		Name:     "John Doe",
		Position: "Manager",
		Salary:   120000.0,
	}
	CreateEmployeeHelper(t, &newEmp)

	// Update data
	updatedEmp := Employee{
		ID:       newEmp.ID,
		Name:     "Jane Doe",
		Position: "Senior Designer",
		Salary:   110000.0,
	}

	// Prepare request
	updatedEmpJSON, _ := json.Marshal(updatedEmp)

	// Create a test server using your router
	router := mux.NewRouter()
	router.HandleFunc("/employees/{id}", UpdateEmployee).Methods("PUT")
	router.Use(loggingMiddleware)

	ts := httptest.NewServer(router)
	defer ts.Close()

	req, err := http.NewRequest("PUT", ts.URL+fmt.Sprintf("/employees/%d", newEmp.ID), bytes.NewBuffer(updatedEmpJSON))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Create a client to send the request through the server
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Error sending request: %v", err)
	}
	defer resp.Body.Close()

	// Validate response
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status %v, got %v", http.StatusOK, resp.StatusCode)
	}

	var returnedEmp Employee
	if err := json.NewDecoder(resp.Body).Decode(&returnedEmp); err != nil {
		t.Fatal("Failed to unmarshal response:", err)
	}
	if returnedEmp != updatedEmp {
		t.Errorf("Updated employee data does not match: got %v, want %v", returnedEmp, updatedEmp)
	}
}

func TestDeleteEmployee(t *testing.T) {

	mutex.Lock()
	employees = make(map[int]*Employee)
	lastID = 0
	mutex.Unlock()

	newEmp := Employee{
		Name:     "John Doe",
		Position: "Manager",
		Salary:   120000.0,
	}

	CreateEmployeeHelper(t, &newEmp)

	// Create a test server using your router
	router := mux.NewRouter()
	router.HandleFunc("/employees/{id}", DeleteEmployee).Methods("DELETE")
	router.Use(loggingMiddleware)

	ts := httptest.NewServer(router)
	defer ts.Close()
	req, err := http.NewRequest("DELETE", ts.URL+fmt.Sprintf("/employees/%d", newEmp.ID), nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a client to send the request through the server
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Error sending request: %v", err)
	}
	defer resp.Body.Close()

	// Validate response
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("Expected status %v, got %v", http.StatusNoContent, resp.StatusCode)
	}

	// Check if employee is actually deleted
	mutex.RLock()
	_, exists := employees[newEmp.ID]
	mutex.RUnlock()
	if exists {
		t.Error("Employee was not deleted")
	}
}

// Helper function to create an employee and add to the map
func CreateEmployeeHelper(t *testing.T, emp *Employee) {
	mutex.Lock()
	defer mutex.Unlock()
	lastID++
	emp.ID = lastID
	employees[emp.ID] = emp
}

func TestListEmployeesPagination(t *testing.T) {

	// Test cases
	testCases := []struct {
		page, pageSize, expectedCount int
	}{
		{1, 10, 10},
		{2, 10, 10},
		{3, 10, 5},  // Last page
		{1, 25, 25}, // All on one page
	}

	for _, tc := range testCases {
		// Reset employee data before each test case
		mutex.Lock()
		employees = make(map[int]*Employee)
		lastID = 0
		mutex.Unlock()

		// Setup: Populate employees
		for i := 0; i < 25; i++ {
			newEmp := Employee{
				Name:     fmt.Sprintf("Employee %d", i),
				Position: "Position",
				Salary:   60000.0,
			}

			mutex.Lock()
			lastID++
			newEmp.ID = lastID
			employees[newEmp.ID] = &newEmp
			mutex.Unlock()
		}

		// Prepare request
		req := httptest.NewRequest("GET", fmt.Sprintf("/employees?page=%d&pageSize=%d", tc.page, tc.pageSize), nil)
		w := httptest.NewRecorder()

		// Execute
		ListEmployees(w, req)

		// Validate response
		if w.Code != http.StatusOK {
			t.Errorf("Expected status %v, got %v", http.StatusOK, w.Code)
		}

		var listedEmps []*Employee
		if err := json.Unmarshal(w.Body.Bytes(), &listedEmps); err != nil {
			t.Fatal("Failed to unmarshal response:", err)
		}

		if len(listedEmps) != tc.expectedCount {
			t.Errorf("Expected %d employees, got %d (page: %d, pageSize: %d)", tc.expectedCount, len(listedEmps), tc.page, tc.pageSize)
		}
	}
}
