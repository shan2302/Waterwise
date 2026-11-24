// Package handler is the single Vercel serverless entry point.
// All /api/* routes are rewritten to this file via vercel.json.
package handler

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
	"sync"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// ── Shared init state ──────────────────────────────────────────────────────────

var (
	initLock   sync.Mutex
	appHandler http.Handler
)

// Handler is the exported entry point Vercel calls for every /api/* request.
func Handler(w http.ResponseWriter, r *http.Request) {
	if err := ensureInitialized(); err != nil {
		log.Printf("initialize: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "Database unavailable. Set DATABASE_URL in your Vercel environment variables.",
		})
		return
	}
	appHandler.ServeHTTP(w, r)
}

func ensureInitialized() error {
	initLock.Lock()
	defer initLock.Unlock()
	if appHandler != nil {
		return nil
	}
	db, err := openDB()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := initializeDB(ctx, db); err != nil {
		db.Close()
		return err
	}
	appHandler = newHandler(db)
	return nil
}

// ── Database ───────────────────────────────────────────────────────────────────

func openDB() (*sql.DB, error) {
	dsn := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("DB_DSN"))
	}
	if dsn == "" {
		u := url.URL{
			Scheme: "postgres",
			User:   url.UserPassword(envOr("DB_USERNAME", "water_app"), os.Getenv("DB_PASSWORD")),
			Host:   net.JoinHostPort(envOr("DB_HOST", "127.0.0.1"), envOr("DB_PORT", "5432")),
			Path:   "/" + envOr("DB_NAME", "water_demand_db"),
			RawQuery: url.Values{
				"sslmode": {envOr("DB_SSLMODE", "disable")},
			}.Encode(),
		}
		dsn = u.String()
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
	q := parsed.Query()
	q.Del("channel_binding")
	parsed.RawQuery = q.Encode()
	return parsed.String(), nil
}

func initializeDB(ctx context.Context, db *sql.DB) error {
	for _, stmt := range []string{
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
	} {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
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
	placeholders := make([]string, 0, 30)
	args := make([]any, 0, 30*5)
	for day := 0; day < 30; day++ {
		date := start.AddDate(0, 0, day)
		temp := 25 + float64(day%9)*1.2
		rain := 0.0
		if day%6 == 0 {
			rain = 12
		} else if day%3 == 0 {
			rain = 3
		}
		occ := 86 + day*7%31
		cons := math.Round(4100 + float64(occ)*13 + temp*38 - rain*16 + float64(day%5)*95)
		n := day * 5
		placeholders = append(placeholders, fmt.Sprintf("($%d,$%d,$%d,$%d,$%d)", n+1, n+2, n+3, n+4, n+5))
		args = append(args, date, cons, temp, rain, occ)
	}
	q := "INSERT INTO water_usage (usage_date, consumption_litres, temperature, rainfall, occupancy) VALUES " +
		strings.Join(placeholders, ",")
	if _, err := tx.ExecContext(ctx, q, args...); err != nil {
		return err
	}
	return tx.Commit()
}

func dateOnly(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

// ── HTTP handler ───────────────────────────────────────────────────────────────

type server struct {
	db  *sql.DB
	now func() time.Time
}

func newHandler(db *sql.DB) http.Handler {
	s := &server{db: db, now: time.Now}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/water-usage/summary", s.handleSummary)
	mux.HandleFunc("/api/water-usage", s.handleWaterUsage)
	mux.HandleFunc("/api/predictions/latest", s.handleLatestPrediction)
	mux.HandleFunc("/api/predictions", s.handlePredictions)
	return corsMiddleware(mux)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if isAllowedOrigin(origin) {
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

func isAllowedOrigin(origin string) bool {
	switch {
	case origin == "http://localhost:5173", origin == "http://127.0.0.1:5173":
		return true
	case strings.HasSuffix(origin, ".vercel.app"):
		return true
	default:
		allowed := strings.TrimSpace(os.Getenv("ALLOWED_ORIGIN"))
		return allowed != "" && origin == allowed
	}
}

// ── Types ──────────────────────────────────────────────────────────────────────

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

type summaryResult struct {
	TotalConsumption   float64 `json:"totalConsumption"`
	AverageConsumption float64 `json:"averageConsumption"`
	HighestConsumption float64 `json:"highestConsumption"`
	LowestConsumption  float64 `json:"lowestConsumption"`
	RecordCount        int64   `json:"recordCount"`
}

type noHistoryError struct{}

func (noHistoryError) Error() string {
	return "No historical water usage data is available for prediction."
}

// ── Handlers ───────────────────────────────────────────────────────────────────

func (s *server) handleWaterUsage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/water-usage" {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		rows, err := s.db.QueryContext(r.Context(),
			`SELECT id, usage_date, consumption_litres, temperature, rainfall, occupancy
			 FROM water_usage ORDER BY usage_date ASC, id ASC`)
		if err != nil {
			s.dbError(w, err)
			return
		}
		defer rows.Close()
		result := make([]waterUsage, 0)
		for rows.Next() {
			var item waterUsage
			var d time.Time
			if err := rows.Scan(&item.ID, &d, &item.ConsumptionLitres, &item.Temperature, &item.Rainfall, &item.Occupancy); err != nil {
				s.dbError(w, err)
				return
			}
			item.UsageDate = d.Format("2006-01-02")
			result = append(result, item)
		}
		if err := rows.Err(); err != nil {
			s.dbError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	case http.MethodPost:
		var input waterUsageInput
		if !decodeJSON(w, r, &input) {
			return
		}
		date, msg := validateUsage(input)
		if msg != "" {
			writeError(w, http.StatusBadRequest, msg)
			return
		}
		parsedDate, _ := time.Parse("2006-01-02", date)
		var id int64
		err := s.db.QueryRowContext(r.Context(),
			`INSERT INTO water_usage (usage_date, consumption_litres, temperature, rainfall, occupancy)
			 VALUES ($1,$2,$3,$4,$5) RETURNING id`,
			parsedDate, *input.ConsumptionLitres, *input.Temperature, *input.Rainfall, *input.Occupancy).Scan(&id)
		if err != nil {
			s.dbError(w, err)
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

func (s *server) handleSummary(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/water-usage/summary" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	var res summaryResult
	err := s.db.QueryRowContext(r.Context(),
		`SELECT COALESCE(SUM(consumption_litres),0),
		        COALESCE(AVG(consumption_litres),0),
		        COALESCE(MAX(consumption_litres),0),
		        COALESCE(MIN(consumption_litres),0),
		        COUNT(*)
		 FROM water_usage`).Scan(
		&res.TotalConsumption, &res.AverageConsumption,
		&res.HighestConsumption, &res.LowestConsumption, &res.RecordCount)
	if err != nil {
		s.dbError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *server) handlePredictions(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/predictions" {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		rows, err := s.db.QueryContext(r.Context(),
			`SELECT id, prediction_date, predicted_demand_litres, demand_status, recommendation, created_at
			 FROM predictions ORDER BY created_at DESC, id DESC`)
		if err != nil {
			s.dbError(w, err)
			return
		}
		defer rows.Close()
		result := make([]prediction, 0)
		for rows.Next() {
			item, err := scanPrediction(rows)
			if err != nil {
				s.dbError(w, err)
				return
			}
			result = append(result, item)
		}
		if err := rows.Err(); err != nil {
			s.dbError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	case http.MethodPost:
		var input predictionInput
		if !decodeJSON(w, r, &input) {
			return
		}
		date, msg := validatePrediction(input)
		if msg != "" {
			writeError(w, http.StatusBadRequest, msg)
			return
		}
		res, err := s.createPrediction(r.Context(), date, *input.Temperature, *input.Rainfall, *input.Occupancy)
		if err != nil {
			var nh noHistoryError
			if errors.As(err, &nh) {
				writeError(w, http.StatusUnprocessableEntity, nh.Error())
				return
			}
			s.dbError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, res)
	default:
		methodNotAllowed(w)
	}
}

func (s *server) handleLatestPrediction(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/predictions/latest" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	item, err := scanPrediction(s.db.QueryRowContext(r.Context(),
		`SELECT id, prediction_date, predicted_demand_litres, demand_status, recommendation, created_at
		 FROM predictions ORDER BY created_at DESC, id DESC LIMIT 1`))
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "No predictions have been generated yet.")
		return
	}
	if err != nil {
		s.dbError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// ── Prediction logic ───────────────────────────────────────────────────────────

func (s *server) createPrediction(ctx context.Context, date string, temperature, rainfall float64, occupancy int) (prediction, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, usage_date, consumption_litres, temperature, rainfall, occupancy FROM water_usage`)
	if err != nil {
		return prediction{}, err
	}
	var records []waterUsage
	for rows.Next() {
		var item waterUsage
		var d time.Time
		if err := rows.Scan(&item.ID, &d, &item.ConsumptionLitres, &item.Temperature, &item.Rainfall, &item.Occupancy); err != nil {
			rows.Close()
			return prediction{}, err
		}
		item.UsageDate = d.Format("2006-01-02")
		records = append(records, item)
	}
	if err := rows.Close(); err != nil {
		return prediction{}, err
	}
	if len(records) == 0 {
		return prediction{}, noHistoryError{}
	}

	estimate := estimateDemand(records, date, temperature, rainfall, occupancy)
	var avg float64
	for _, rec := range records {
		avg += rec.ConsumptionLitres
	}
	avg /= float64(len(records))
	if !isFinite(estimate) || estimate <= 0 {
		estimate = baseline(records, avg, temperature, occupancy)
	}
	if estimate < 0 {
		estimate = 0
	}
	estimate = math.Round(estimate*100) / 100

	status := "NORMAL"
	recommendation := "Demand is within the usual historical range. Continue avoiding unnecessary water use and review consumption regularly."
	if estimate > avg*1.10 {
		status = "HIGH"
		recommendation = "Expected demand is above the normal historical range. Review planned use, check for possible leaks, and reduce non-essential water use."
	}

	createdAt := s.now()
	parsedDate, _ := time.Parse("2006-01-02", date)
	var id int64
	err = s.db.QueryRowContext(ctx,
		`INSERT INTO predictions (prediction_date, predicted_demand_litres, demand_status, recommendation, created_at)
		 VALUES ($1,$2,$3,$4,$5) RETURNING id`,
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

func estimateDemand(records []waterUsage, date string, temperature, rainfall float64, occupancy int) float64 {
	avg := 0.0
	for _, r := range records {
		avg += r.ConsumptionLitres
	}
	avg /= float64(len(records))
	if len(records) < 8 {
		return baseline(records, avg, temperature, occupancy)
	}
	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return math.NaN()
	}
	const dims = 6
	matrix := make([][]float64, dims)
	for i := range matrix {
		matrix[i] = make([]float64, dims+1)
	}
	for _, rec := range records {
		rd, err := time.Parse("2006-01-02", rec.UsageDate)
		if err != nil {
			return math.NaN()
		}
		addRegressionRow(matrix,
			features(rec.Temperature, rec.Rainfall, float64(rec.Occupancy), int(rd.Month()), isoWeekday(rd)),
			rec.ConsumptionLitres)
	}
	for i := 1; i < dims; i++ {
		matrix[i][i] += 1e-6
	}
	coefficients, ok := solveRegression(matrix)
	if !ok {
		return baseline(records, avg, temperature, occupancy)
	}
	input := features(temperature, rainfall, float64(occupancy), int(parsedDate.Month()), isoWeekday(parsedDate))
	estimate := 0.0
	for i, c := range coefficients {
		estimate += c * input[i]
	}
	return estimate
}

func baseline(records []waterUsage, average, temperature float64, occupancy int) float64 {
	occAvg, tempAvg := 0.0, 0.0
	for _, r := range records {
		occAvg += float64(r.Occupancy)
		tempAvg += r.Temperature
	}
	n := float64(len(records))
	occAvg /= n
	tempAvg /= n
	occFactor := clamp(float64(occupancy)/math.Max(1, occAvg), 0.5, 1.8)
	tempFactor := clamp(1+(temperature-tempAvg)*0.005, 0.9, 1.1)
	return average * occFactor * tempFactor
}

func features(temperature, rainfall, occupancy float64, month, weekday int) []float64 {
	return []float64{1, temperature, rainfall, occupancy, float64(month), float64(weekday)}
}

func isoWeekday(d time.Time) int { return (int(d.Weekday())+6)%7 + 1 }

func addRegressionRow(m [][]float64, row []float64, target float64) {
	for i := range row {
		for j := range row {
			m[i][j] += row[i] * row[j]
		}
		m[i][len(row)] += row[i] * target
	}
}

func solveRegression(aug [][]float64) ([]float64, bool) {
	n := len(aug)
	for pivot := 0; pivot < n; pivot++ {
		best := pivot
		for row := pivot + 1; row < n; row++ {
			if math.Abs(aug[row][pivot]) > math.Abs(aug[best][pivot]) {
				best = row
			}
		}
		aug[pivot], aug[best] = aug[best], aug[pivot]
		if math.Abs(aug[pivot][pivot]) < 1e-10 {
			return nil, false
		}
		div := aug[pivot][pivot]
		for col := pivot; col <= n; col++ {
			aug[pivot][col] /= div
		}
		for row := 0; row < n; row++ {
			if row == pivot {
				continue
			}
			f := aug[row][pivot]
			for col := pivot; col <= n; col++ {
				aug[row][col] -= f * aug[pivot][col]
			}
		}
	}
	result := make([]float64, n)
	for i := range result {
		result[i] = aug[i][n]
	}
	return result, true
}

func clamp(v, lo, hi float64) float64  { return math.Max(lo, math.Min(hi, v)) }
func isFinite(v float64) bool          { return !math.IsNaN(v) && !math.IsInf(v, 0) }

// ── Validation ─────────────────────────────────────────────────────────────────

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

// ── Scan helper ────────────────────────────────────────────────────────────────

type rowScanner interface{ Scan(dest ...any) error }

func scanPrediction(row rowScanner) (prediction, error) {
	var item prediction
	var d, createdAt time.Time
	if err := row.Scan(&item.ID, &d, &item.PredictedDemandLitres, &item.DemandStatus, &item.Recommendation, &createdAt); err != nil {
		return item, err
	}
	item.PredictionDate = d.Format("2006-01-02")
	item.CreatedAt = createdAt.Format("2006-01-02T15:04:05")
	return item, nil
}

// ── HTTP utils ─────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON request body.")
		return false
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "Invalid JSON request body.")
		return false
	}
	return true
}

func methodNotAllowed(w http.ResponseWriter) {
	w.Header().Set("Allow", "GET, POST, OPTIONS")
	writeError(w, http.StatusMethodNotAllowed, "Method not allowed.")
}

func (s *server) dbError(w http.ResponseWriter, err error) {
	log.Printf("database error: %v", err)
	writeError(w, http.StatusServiceUnavailable, "Database unavailable.")
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
