package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func BenchmarkGetProducts(b *testing.B) {
	// Create a new mock database
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		b.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockDB.Close()

	// Replace the global db variable with the mock
	oldDB := db
	db = mockDB
	defer func() { db = oldDB }()

	// Prepare the request
	req, _ := http.NewRequest("GET", "/products", nil)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Create rows for the mock - we need fresh rows for each iteration as they are consumed
		rows := sqlmock.NewRows([]string{"id", "name", "description", "price", "stock", "created_at"})
		for j := 0; j < 1000; j++ {
			rows.AddRow(j, fmt.Sprintf("Product %d", j), "Description", 10.0, 100, time.Now())
		}

		mock.ExpectQuery("SELECT id, name, description, price, stock, created_at FROM products ORDER BY id").
			WillReturnRows(rows)
		b.StartTimer()

		w := httptest.NewRecorder()
		getProducts(w, req)

		// Verify expectations were met (optional, but good sanity check)
		// if err := mock.ExpectationsWereMet(); err != nil {
		// 	b.Errorf("there were unfulfilled expectations: %s", err)
		// }
	}
}

func TestGetProducts(t *testing.T) {
	// Create a new mock database
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockDB.Close()

	// Replace the global db variable with the mock
	oldDB := db
	db = mockDB
	defer func() { db = oldDB }()

	rows := sqlmock.NewRows([]string{"id", "name", "description", "price", "stock", "created_at"}).
		AddRow(1, "Test Product", "Test Description", 10.0, 100, time.Now())

	mock.ExpectQuery("SELECT id, name, description, price, stock, created_at FROM products ORDER BY id").
		WillReturnRows(rows)

	req, _ := http.NewRequest("GET", "/products", nil)
	w := httptest.NewRecorder()

	getProducts(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status OK, got %v", w.Code)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
