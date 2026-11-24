package backend

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAPIListsUsageAndSummary(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectQuery("SELECT id, usage_date, consumption_litres, temperature, rainfall, occupancy FROM water_usage").
		WillReturnRows(sqlmock.NewRows([]string{"id", "usage_date", "consumption_litres", "temperature", "rainfall", "occupancy"}).
			AddRow(1, time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), 5000.0, 28.0, 2.0, 100))
	mock.ExpectQuery("SELECT COALESCE\\(SUM\\(consumption_litres\\), 0\\)").
		WillReturnRows(sqlmock.NewRows([]string{"total", "average", "highest", "lowest", "count"}).
			AddRow(5000.0, 5000.0, 5000.0, 5000.0, 1))

	handler := NewHandler(db)
	usageResponse := httptest.NewRecorder()
	handler.ServeHTTP(usageResponse, httptest.NewRequest(http.MethodGet, "/api/water-usage", nil))
	if usageResponse.Code != http.StatusOK {
		t.Fatalf("usage status = %d, want 200: %s", usageResponse.Code, usageResponse.Body)
	}
	var records []waterUsage
	if err := json.Unmarshal(usageResponse.Body.Bytes(), &records); err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].UsageDate != "2026-10-01" {
		t.Fatalf("unexpected usage response: %+v", records)
	}

	summaryResponse := httptest.NewRecorder()
	handler.ServeHTTP(summaryResponse, httptest.NewRequest(http.MethodGet, "/api/water-usage/summary", nil))
	if summaryResponse.Code != http.StatusOK || !strings.Contains(summaryResponse.Body.String(), `"recordCount":1`) {
		t.Fatalf("unexpected summary response: %d %s", summaryResponse.Code, summaryResponse.Body)
	}
	assertDBExpectations(t, mock)
}

func TestAPIAddsUsageAndValidatesInput(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectExec("INSERT INTO water_usage").
		WithArgs("2026-10-01", 5000.0, 28.0, 2.0, 100).
		WillReturnResult(sqlmock.NewResult(7, 1))

	handler := NewHandler(db)
	request := httptest.NewRequest(http.MethodPost, "/api/water-usage",
		strings.NewReader(`{"usageDate":"2026-10-01","consumptionLitres":5000,"temperature":28,"rainfall":2,"occupancy":100}`))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"id":7`) {
		t.Fatalf("unexpected create response: %d %s", response.Code, response.Body)
	}

	invalid := httptest.NewRecorder()
	handler.ServeHTTP(invalid, httptest.NewRequest(http.MethodPost, "/api/water-usage",
		strings.NewReader(`{"usageDate":"2026-10-01","consumptionLitres":0,"temperature":28,"rainfall":2,"occupancy":100}`)))
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid input status = %d, want 400", invalid.Code)
	}
	assertDBExpectations(t, mock)
}

func TestAPIHandlesPredictionHistoryAndLatest(t *testing.T) {
	db, mock := newMockDB(t)
	predictionRows := sqlmock.NewRows([]string{
		"id", "prediction_date", "predicted_demand_litres", "demand_status", "recommendation", "created_at",
	}).AddRow(3, time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC), 5100.0, "NORMAL", "Keep monitoring.", time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC))
	mock.ExpectQuery("SELECT id, prediction_date, predicted_demand_litres, demand_status, recommendation, created_at FROM predictions").
		WillReturnRows(predictionRows)
	mock.ExpectQuery("SELECT id, prediction_date, predicted_demand_litres, demand_status, recommendation, created_at FROM predictions").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "prediction_date", "predicted_demand_litres", "demand_status", "recommendation", "created_at",
		}).AddRow(3, time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC), 5100.0, "NORMAL", "Keep monitoring.", time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)))
	mock.ExpectQuery("SELECT id, prediction_date, predicted_demand_litres, demand_status, recommendation, created_at FROM predictions").
		WillReturnError(sql.ErrNoRows)

	handler := NewHandler(db)
	for _, path := range []string{"/api/predictions", "/api/predictions/latest"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"demandStatus":"NORMAL"`) {
			t.Fatalf("%s response = %d %s", path, response.Code, response.Body)
		}
	}
	notFound := httptest.NewRecorder()
	handler.ServeHTTP(notFound, httptest.NewRequest(http.MethodGet, "/api/predictions/latest", nil))
	if notFound.Code != http.StatusNotFound {
		t.Fatalf("missing latest prediction status = %d, want 404", notFound.Code)
	}
	assertDBExpectations(t, mock)
}

func TestAPIPredictionGeneration(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectQuery("SELECT id, usage_date, consumption_litres, temperature, rainfall, occupancy FROM water_usage").
		WillReturnRows(sqlmock.NewRows([]string{"id", "usage_date", "consumption_litres", "temperature", "rainfall", "occupancy"}).
			AddRow(1, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), 5000.0, 25.0, 0.0, 100).
			AddRow(2, time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC), 6000.0, 27.0, 2.0, 120))
	mock.ExpectExec("INSERT INTO predictions").
		WithArgs("2026-10-03", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(12, 1))

	handler := NewHandler(db)
	request := httptest.NewRequest(http.MethodPost, "/api/predictions",
		strings.NewReader(`{"predictionDate":"2026-10-03","temperature":31,"rainfall":0,"occupancy":150}`))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"id":12`) {
		t.Fatalf("unexpected prediction response: %d %s", response.Code, response.Body)
	}
	assertDBExpectations(t, mock)
}

func TestCORSPreflightAndMethodNotAllowed(t *testing.T) {
	handler := NewHandler(nil)
	preflight := httptest.NewRequest(http.MethodOptions, "/api/water-usage", nil)
	preflight.Header.Set("Origin", "http://localhost:5173")
	preflightResponse := httptest.NewRecorder()
	handler.ServeHTTP(preflightResponse, preflight)
	if preflightResponse.Code != http.StatusNoContent || preflightResponse.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
		t.Fatalf("unexpected preflight response: %d %+v", preflightResponse.Code, preflightResponse.Header())
	}

	methodResponse := httptest.NewRecorder()
	handler.ServeHTTP(methodResponse, httptest.NewRequest(http.MethodDelete, "/api/water-usage", nil))
	if methodResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("DELETE status = %d, want 405", methodResponse.Code)
	}
}

func newMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Close()
	})
	return db, mock
}

func assertDBExpectations(t *testing.T, mock sqlmock.Sqlmock) {
	t.Helper()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
