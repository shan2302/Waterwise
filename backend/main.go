package backend

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type waterUsage struct {
	ID                int64   `json:"id"`
	UsageDate         string  `json:"usageDate"`
	ConsumptionLitres float64 `json:"consumptionLitres"`
	Temperature       float64 `json:"temperature"`
	Rainfall          float64 `json:"rainfall"`
	Occupancy         int     `json:"occupancy"`
}

type waterUsageInput struct {
	UsageDate         string   `json:"usageDate"`
	ConsumptionLitres *float64 `json:"consumptionLitres"`
	Temperature       *float64 `json:"temperature"`
	Rainfall          *float64 `json:"rainfall"`
	Occupancy         *int     `json:"occupancy"`
}

type predictionInput struct {
	PredictionDate string   `json:"predictionDate"`
	Temperature    *float64 `json:"temperature"`
	Rainfall       *float64 `json:"rainfall"`
	Occupancy      *int     `json:"occupancy"`
}

type prediction struct {
	ID                    int64   `json:"id"`
	PredictionDate        string  `json:"predictionDate"`
	PredictedDemandLitres float64 `json:"predictedDemandLitres"`
	DemandStatus          string  `json:"demandStatus"`
	Recommendation        string  `json:"recommendation"`
	CreatedAt             string  `json:"createdAt"`
}

type summary struct {
	TotalConsumption   float64 `json:"totalConsumption"`
	AverageConsumption float64 `json:"averageConsumption"`
	HighestConsumption float64 `json:"highestConsumption"`
	LowestConsumption  float64 `json:"lowestConsumption"`
	RecordCount        int64   `json:"recordCount"`
}

type api struct {
	db  *sql.DB
	now func() time.Time
}

func OpenDB() (*sql.DB, error) {
	dsn := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("DB_DSN"))
	}
	if dsn == "" {
		connectionURL := url.URL{
			Scheme: "postgres",
			User:   url.UserPassword(envOr("DB_USERNAME", "water_app"), os.Getenv("DB_PASSWORD")),
			Host:   net.JoinHostPort(envOr("DB_HOST", "127.0.0.1"), envOr("DB_PORT", "5432")),
			Path:   "/" + envOr("DB_NAME", "water_demand_db"),
			RawQuery: url.Values{
				"sslmode": {envOr("DB_SSLMODE", "disable")},
			}.Encode(),
		}
		dsn = connectionURL.String()
	}
	dsn, err := normalizePostgresURL(dsn)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func normalizePostgresURL(raw string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	if parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" {
		return "", errors.New("DATABASE_URL must use the postgres:// or postgresql:// scheme")
	}
	if parsed.Hostname() == "" || strings.Trim(parsed.Path, "/") == "" {
		return "", errors.New("DATABASE_URL must include a host and database name")
	}
	query := parsed.Query()
	query.Del("channel_binding")
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func Initialize(ctx context.Context, db *sql.DB) error {
	if err := initialize(ctx, db); err != nil {
		return err
	}
	return nil
}

func NewHandler(db *sql.DB) http.Handler {
	server := &api{db: db, now: time.Now}
	return server.routes()
}

func Run() error {
	db, err := OpenDB()
	if err != nil {
		return err
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := Initialize(ctx, db); err != nil {
		return err
	}
	addr := ":" + envOr("PORT", "8080")
	log.Printf("water demand API listening on %s", addr)
	return http.ListenAndServe(addr, NewHandler(db))
}

func envOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func initialize(ctx context.Context, db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS water_usage (
			id BIGSERIAL PRIMARY KEY,
			usage_date DATE NOT NULL,
			consumption_litres DOUBLE PRECISION NOT NULL,
			temperature DOUBLE PRECISION NOT NULL,
			rainfall DOUBLE PRECISION NOT NULL,
			occupancy INT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS predictions (
			id BIGSERIAL PRIMARY KEY,
			prediction_date DATE NOT NULL,
			predicted_demand_litres DOUBLE PRECISION NOT NULL,
			demand_status VARCHAR(255) NOT NULL,
			recommendation VARCHAR(500) NOT NULL,
			created_at TIMESTAMPTZ NOT NULL
		)`,
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "LOCK TABLE water_usage IN SHARE ROW EXCLUSIVE MODE"); err != nil {
		return err
	}

	var count int64
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM water_usage").Scan(&count); err != nil {
		return err
	}
	if count != 0 {
		return tx.Commit()
	}

	start := dateOnly(time.Now()).AddDate(0, 0, -29)
	valueStrings := make([]string, 0, 30)
	valueArgs := make([]any, 0, 30*5)
	for day := 0; day < 30; day++ {
		date := start.AddDate(0, 0, day)
		temperature := 25 + float64(day%9)*1.2
		rainfall := 0.0
		if day%6 == 0 {
			rainfall = 12
		} else if day%3 == 0 {
			rainfall = 3
		}
		occupancy := 86 + day*7%31
		consumption := math.Round(4100 + float64(occupancy)*13 + temperature*38 - rainfall*16 + float64(day%5)*95)
		n := day * 5
		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d)", n+1, n+2, n+3, n+4, n+5))
		valueArgs = append(valueArgs, date, consumption, temperature, rainfall, occupancy)
	}
	stmt := fmt.Sprintf("INSERT INTO water_usage (usage_date, consumption_litres, temperature, rainfall, occupancy) VALUES %s", strings.Join(valueStrings, ", "))
	if _, err := tx.ExecContext(ctx, stmt, valueArgs...); err != nil {
		return err
	}
	return tx.Commit()
}

func dateOnly(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, value.Location())
}

func (a *api) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/water-usage/summary", a.handleSummary)
	mux.HandleFunc("/api/water-usage", a.handleWaterUsage)
	mux.HandleFunc("/api/predictions/latest", a.handleLatestPrediction)
	mux.HandleFunc("/api/predictions", a.handlePredictions)
	return cors(mux)
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "http://localhost:5173" || origin == "http://127.0.0.1:5173" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *api) handleWaterUsage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/water-usage" {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		rows, err := a.db.QueryContext(r.Context(),
			`SELECT id, usage_date, consumption_litres, temperature, rainfall, occupancy
			 FROM water_usage ORDER BY usage_date ASC, id ASC`)
		if err != nil {
			a.databaseError(w, err)
			return
		}
		defer rows.Close()
		result := make([]waterUsage, 0)
		for rows.Next() {
			var item waterUsage
			var date time.Time
			if err := rows.Scan(&item.ID, &date, &item.ConsumptionLitres, &item.Temperature, &item.Rainfall, &item.Occupancy); err != nil {
				a.databaseError(w, err)
				return
			}
			item.UsageDate = date.Format("2006-01-02")
			result = append(result, item)
		}
		if err := rows.Err(); err != nil {
			a.databaseError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	case http.MethodPost:
		var input waterUsageInput
		if !decodeJSON(w, r, &input) {
			return
		}
		date, message := validateUsage(input)
		if message != "" {
			writeError(w, http.StatusBadRequest, message)
			return
		}
		parsedDate, _ := time.Parse("2006-01-02", date)
		var id int64
		err := a.db.QueryRowContext(r.Context(),
			`INSERT INTO water_usage (usage_date, consumption_litres, temperature, rainfall, occupancy)
			 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
			parsedDate, *input.ConsumptionLitres, *input.Temperature, *input.Rainfall, *input.Occupancy).Scan(&id)
		if err != nil {
			a.databaseError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, waterUsage{
			ID: id, UsageDate: date,
			ConsumptionLitres: *input.ConsumptionLitres,
			Temperature:       *input.Temperature,
			Rainfall:          *input.Rainfall,
			Occupancy:         *input.Occupancy,
		})
	default:
		methodNotAllowed(w)
	}
}

func (a *api) handleSummary(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/water-usage/summary" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	var result summary
	err := a.db.QueryRowContext(r.Context(),
		`SELECT COALESCE(SUM(consumption_litres), 0),
		 COALESCE(AVG(consumption_litres), 0),
		 COALESCE(MAX(consumption_litres), 0),
		 COALESCE(MIN(consumption_litres), 0),
		 COUNT(*)
		 FROM water_usage`).Scan(
		&result.TotalConsumption, &result.AverageConsumption,
		&result.HighestConsumption, &result.LowestConsumption, &result.RecordCount)
	if err != nil {
		a.databaseError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *api) handlePredictions(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/predictions" {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		rows, err := a.db.QueryContext(r.Context(),
			`SELECT id, prediction_date, predicted_demand_litres, demand_status, recommendation, created_at
			 FROM predictions ORDER BY created_at DESC, id DESC`)
		if err != nil {
			a.databaseError(w, err)
			return
		}
		defer rows.Close()
		result := make([]prediction, 0)
		for rows.Next() {
			item, err := scanPrediction(rows)
			if err != nil {
				a.databaseError(w, err)
				return
			}
			result = append(result, item)
		}
		if err := rows.Err(); err != nil {
			a.databaseError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	case http.MethodPost:
		var input predictionInput
		if !decodeJSON(w, r, &input) {
			return
		}
		date, message := validatePrediction(input)
		if message != "" {
			writeError(w, http.StatusBadRequest, message)
			return
		}
		result, err := a.createPrediction(r.Context(), date, *input.Temperature, *input.Rainfall, *input.Occupancy)
		if err != nil {
			var noHistory noHistoryError
			if errors.As(err, &noHistory) {
				writeError(w, http.StatusUnprocessableEntity, noHistory.Error())
				return
			}
			a.databaseError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	default:
		methodNotAllowed(w)
	}
}

func (a *api) handleLatestPrediction(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/predictions/latest" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	item, err := scanPrediction(a.db.QueryRowContext(r.Context(),
		`SELECT id, prediction_date, predicted_demand_litres, demand_status, recommendation, created_at
		 FROM predictions ORDER BY created_at DESC, id DESC LIMIT 1`))
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "No predictions have been generated yet.")
		return
	}
	if err != nil {
		a.databaseError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanPrediction(row rowScanner) (prediction, error) {
	var item prediction
	var date time.Time
	var createdAt time.Time
	err := row.Scan(&item.ID, &date, &item.PredictedDemandLitres, &item.DemandStatus, &item.Recommendation, &createdAt)
	if err != nil {
		return item, err
	}
	item.PredictionDate = date.Format("2006-01-02")
	item.CreatedAt = createdAt.Format("2006-01-02T15:04:05")
	return item, nil
}

func (a *api) createPrediction(ctx context.Context, date string, temperature, rainfall float64, occupancy int) (prediction, error) {
	rows, err := a.db.QueryContext(ctx,
		`SELECT id, usage_date, consumption_litres, temperature, rainfall, occupancy FROM water_usage`)
	if err != nil {
		return prediction{}, err
	}
	records := make([]waterUsage, 0)
	for rows.Next() {
		var item waterUsage
		var usageDate time.Time
		if err := rows.Scan(&item.ID, &usageDate, &item.ConsumptionLitres, &item.Temperature, &item.Rainfall, &item.Occupancy); err != nil {
			rows.Close()
			return prediction{}, err
		}
		item.UsageDate = usageDate.Format("2006-01-02")
		records = append(records, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return prediction{}, err
	}
	if err := rows.Close(); err != nil {
		return prediction{}, err
	}
	if len(records) == 0 {
		return prediction{}, noHistoryError{}
	}

	estimate := estimateDemand(records, date, temperature, rainfall, occupancy)
	average := 0.0
	for _, record := range records {
		average += record.ConsumptionLitres
	}
	average /= float64(len(records))
	if !isFinite(estimate) || estimate <= 0 {
		estimate = baseline(records, average, temperature, occupancy)
	}
	if estimate < 0 {
		estimate = 0
	}
	estimate = math.Round(estimate*100) / 100

	status := "NORMAL"
	recommendation := "Demand is within the usual historical range. Continue avoiding unnecessary water use and review consumption regularly."
	if estimate > average*1.10 {
		status = "HIGH"
		recommendation = "Expected demand is above the normal historical range. Review planned use, check for possible leaks, and reduce non-essential water use."
	}

	createdAt := a.now()
	parsedDate, _ := time.Parse("2006-01-02", date)
	var id int64
	err = a.db.QueryRowContext(ctx,
		`INSERT INTO predictions (prediction_date, predicted_demand_litres, demand_status, recommendation, created_at)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		parsedDate, estimate, status, recommendation, createdAt).Scan(&id)
	if err != nil {
		return prediction{}, err
	}
	return prediction{
		ID: id, PredictionDate: date, PredictedDemandLitres: estimate,
		DemandStatus: status, Recommendation: recommendation,
		CreatedAt: createdAt.Format("2006-01-02T15:04:05"),
	}, nil
}

type noHistoryError struct{}

func (noHistoryError) Error() string {
	return "No historical water usage data is available for prediction."
}

func estimateDemand(records []waterUsage, date string, temperature, rainfall float64, occupancy int) float64 {
	average := 0.0
	for _, record := range records {
		average += record.ConsumptionLitres
	}
	average /= float64(len(records))
	if len(records) < 8 {
		return baseline(records, average, temperature, occupancy)
	}
	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return math.NaN()
	}
	const dimensions = 6
	normal := make([][]float64, dimensions)
	for i := range normal {
		normal[i] = make([]float64, dimensions+1)
	}
	for _, record := range records {
		recordDate, err := time.Parse("2006-01-02", record.UsageDate)
		if err != nil {
			return math.NaN()
		}
		addRegressionRow(normal,
			features(record.Temperature, record.Rainfall, float64(record.Occupancy), int(recordDate.Month()), isoWeekday(recordDate)),
			record.ConsumptionLitres)
	}
	for i := 1; i < dimensions; i++ {
		normal[i][i] += 1e-6
	}
	coefficients, ok := solveRegression(normal)
	if !ok {
		return baseline(records, average, temperature, occupancy)
	}
	input := features(temperature, rainfall, float64(occupancy), int(parsedDate.Month()), isoWeekday(parsedDate))
	estimate := 0.0
	for i := range coefficients {
		estimate += coefficients[i] * input[i]
	}
	return estimate
}

func baseline(records []waterUsage, average, temperature float64, occupancy int) float64 {
	occupancyAverage, temperatureAverage := 0.0, 0.0
	for _, record := range records {
		occupancyAverage += float64(record.Occupancy)
		temperatureAverage += record.Temperature
	}
	occupancyAverage /= float64(len(records))
	temperatureAverage /= float64(len(records))
	occupancyFactor := clamp(float64(occupancy)/math.Max(1, occupancyAverage), 0.5, 1.8)
	temperatureFactor := clamp(1+(temperature-temperatureAverage)*0.005, 0.9, 1.1)
	return average * occupancyFactor * temperatureFactor
}

func features(temperature, rainfall, occupancy float64, month, weekday int) []float64 {
	return []float64{1, temperature, rainfall, occupancy, float64(month), float64(weekday)}
}

func isoWeekday(date time.Time) int {
	return (int(date.Weekday())+6)%7 + 1
}

func addRegressionRow(matrix [][]float64, row []float64, target float64) {
	for i := range row {
		for j := range row {
			matrix[i][j] += row[i] * row[j]
		}
		matrix[i][len(row)] += row[i] * target
	}
}

func solveRegression(augmented [][]float64) ([]float64, bool) {
	n := len(augmented)
	for pivot := 0; pivot < n; pivot++ {
		best := pivot
		for row := pivot + 1; row < n; row++ {
			if math.Abs(augmented[row][pivot]) > math.Abs(augmented[best][pivot]) {
				best = row
			}
		}
		augmented[pivot], augmented[best] = augmented[best], augmented[pivot]
		if math.Abs(augmented[pivot][pivot]) < 1e-10 {
			return nil, false
		}
		divisor := augmented[pivot][pivot]
		for col := pivot; col <= n; col++ {
			augmented[pivot][col] /= divisor
		}
		for row := 0; row < n; row++ {
			if row == pivot {
				continue
			}
			factor := augmented[row][pivot]
			for col := pivot; col <= n; col++ {
				augmented[row][col] -= factor * augmented[pivot][col]
			}
		}
	}
	result := make([]float64, n)
	for row := range result {
		result[row] = augmented[row][n]
	}
	return result, true
}

func clamp(value, min, max float64) float64 {
	return math.Max(min, math.Min(max, value))
}

func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func validateUsage(input waterUsageInput) (string, string) {
	date, err := parseDate(input.UsageDate)
	if err != nil {
		return "", "usageDate: must be a valid date"
	}
	if input.ConsumptionLitres == nil {
		return "", "consumptionLitres: must not be null"
	}
	if !isFinite(*input.ConsumptionLitres) || *input.ConsumptionLitres <= 0 || *input.ConsumptionLitres > 100000000 {
		return "", "consumptionLitres: must be greater than 0 and no greater than 100000000"
	}
	if input.Temperature == nil {
		return "", "temperature: must not be null"
	}
	if !isFinite(*input.Temperature) || *input.Temperature < -20 || *input.Temperature > 60 {
		return "", "temperature: must be between -20 and 60"
	}
	if input.Rainfall == nil {
		return "", "rainfall: must not be null"
	}
	if !isFinite(*input.Rainfall) || *input.Rainfall < 0 || *input.Rainfall > 1000 {
		return "", "rainfall: must be between 0 and 1000"
	}
	if input.Occupancy == nil {
		return "", "occupancy: must not be null"
	}
	if *input.Occupancy < 1 || *input.Occupancy > 1000000 {
		return "", "occupancy: must be between 1 and 1000000"
	}
	return date, ""
}

func validatePrediction(input predictionInput) (string, string) {
	date, err := parseDate(input.PredictionDate)
	if err != nil {
		return "", "predictionDate: must be a valid date"
	}
	if input.Temperature == nil {
		return "", "temperature: must not be null"
	}
	if !isFinite(*input.Temperature) || *input.Temperature < -20 || *input.Temperature > 60 {
		return "", "temperature: must be between -20 and 60"
	}
	if input.Rainfall == nil {
		return "", "rainfall: must not be null"
	}
	if !isFinite(*input.Rainfall) || *input.Rainfall < 0 || *input.Rainfall > 1000 {
		return "", "rainfall: must be between 0 and 1000"
	}
	if input.Occupancy == nil {
		return "", "occupancy: must not be null"
	}
	if *input.Occupancy < 1 || *input.Occupancy > 1000000 {
		return "", "occupancy: must be between 1 and 1000000"
	}
	return date, ""
}

func parseDate(value string) (string, error) {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil || parsed.Format("2006-01-02") != value {
		return "", errors.New("invalid date")
	}
	return value, nil
}

func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(destination); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON request body.")
		return false
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "Invalid JSON request body.")
		return false
	}
	return true
}

func methodNotAllowed(w http.ResponseWriter) {
	w.Header().Set("Allow", "GET, POST, OPTIONS")
	writeError(w, http.StatusMethodNotAllowed, "Method not allowed.")
}

func (a *api) databaseError(w http.ResponseWriter, err error) {
	log.Printf("database operation failed: %v", err)
	writeError(w, http.StatusServiceUnavailable, "Database unavailable. Check that the PostgreSQL database is reachable and configured.")
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("write JSON response: %v", err)
	}
}
