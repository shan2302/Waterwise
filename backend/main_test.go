package backend

import (
	"math"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestNormalizePostgresURL(t *testing.T) {
	connectionURL, err := normalizePostgresURL("postgresql://water_app:p%40ss@db.example.com:5432/water_demand_db?sslmode=require&channel_binding=require")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(connectionURL)
	if err != nil {
		t.Fatal(err)
	}
	password, _ := parsed.User.Password()
	if parsed.User.Username() != "water_app" || password != "p@ss" || strings.Trim(parsed.Path, "/") != "water_demand_db" {
		t.Fatal("database URL credentials or database name were not preserved")
	}
	if parsed.Host != "db.example.com:5432" || parsed.Query().Get("sslmode") != "require" || parsed.Query().Has("channel_binding") {
		t.Fatal("database URL connection options were not normalized correctly")
	}
	if _, err := pgx.ParseConfig(connectionURL); err != nil {
		t.Fatalf("pgx rejected the normalized database URL: %v", err)
	}
	if _, err := normalizePostgresURL("mysql://user:pass@db.example.com/database"); err == nil {
		t.Fatal("expected MySQL URL to be rejected")
	}
}

func TestValidateUsage(t *testing.T) {
	consumption, temperature, rainfall, occupancy := 5000.0, 28.0, 2.0, 100
	input := waterUsageInput{
		UsageDate:         "2026-10-01",
		ConsumptionLitres: &consumption,
		Temperature:       &temperature,
		Rainfall:          &rainfall,
		Occupancy:         &occupancy,
	}
	date, message := validateUsage(input)
	if message != "" || date != input.UsageDate {
		t.Fatalf("valid input rejected: date=%q message=%q", date, message)
	}

	consumption = 0
	_, message = validateUsage(input)
	if message == "" {
		t.Fatal("expected zero consumption to be rejected")
	}
}

func TestValidatePredictionRejectsInvalidDateAndOccupancy(t *testing.T) {
	temperature, rainfall, occupancy := 25.0, 0.0, 1
	input := predictionInput{
		PredictionDate: "2026-02-30",
		Temperature:    &temperature,
		Rainfall:       &rainfall,
		Occupancy:      &occupancy,
	}
	if _, message := validatePrediction(input); message == "" {
		t.Fatal("expected impossible date to be rejected")
	}
	input.PredictionDate = "2026-10-03"
	occupancy = 1000001
	if _, message := validatePrediction(input); message == "" {
		t.Fatal("expected out-of-range occupancy to be rejected")
	}
}

func TestEstimateDemandRegression(t *testing.T) {
	records := make([]waterUsage, 0, 1000)
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for day := 0; day < 1000; day++ {
		date := start.AddDate(0, 0, day)
		temperature := 18 + float64(day*17%230)/10
		rainfall := float64(day*11%100) / 10
		occupancy := 50 + day*37%600
		consumption := 4000 + temperature*20 + rainfall*10 + float64(occupancy)*5 +
			float64(date.Month())*30 + float64(isoWeekday(date))*10
		records = append(records, waterUsage{
			UsageDate:         date.Format("2006-01-02"),
			ConsumptionLitres: consumption,
			Temperature:       temperature,
			Rainfall:          rainfall,
			Occupancy:         occupancy,
		})
	}

	forecastDate := "2026-10-03"
	temperature, rainfall, occupancy := 31.0, 4.0, 140
	parsedDate, _ := time.Parse("2006-01-02", forecastDate)
	expected := 4000 + temperature*20 + rainfall*10 + float64(occupancy)*5 +
		float64(parsedDate.Month())*30 + float64(isoWeekday(parsedDate))*10
	actual := estimateDemand(records, forecastDate, temperature, rainfall, occupancy)
	if math.Abs(actual-expected) > 0.01 {
		t.Fatalf("regression estimate = %.4f, want %.4f", actual, expected)
	}
}

func TestBaselineEstimate(t *testing.T) {
	records := []waterUsage{
		{ConsumptionLitres: 4000, Temperature: 20, Occupancy: 80},
		{ConsumptionLitres: 6000, Temperature: 30, Occupancy: 120},
	}
	actual := baseline(records, 5000, 25, 100)
	if actual != 5000 {
		t.Fatalf("baseline = %.2f, want 5000", actual)
	}
}

func TestWeekdayUsesISOConvention(t *testing.T) {
	sunday := time.Date(2026, 10, 4, 0, 0, 0, 0, time.UTC)
	if weekday := isoWeekday(sunday); weekday != 7 {
		t.Fatalf("Sunday weekday = %d, want 7", weekday)
	}
}
