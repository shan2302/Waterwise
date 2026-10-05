# AI-Based Water Demand Prediction and Conservation System

## 1. Project Overview

This project is a local web application for estimating future water demand. It combines saved water-use records with weather, occupancy, and date inputs. It then shows an estimated demand, a NORMAL or HIGH status, and a conservation suggestion.

The system is a decision-support prototype. It does not control water supply. **This prototype does not require any physical hardware. It uses historical/sample data and runs locally on a computer.** The included records are sample data for demonstration, not measurements collected by the project team.

The dashboard also shows verified, city-wide water-supply figures for the **Bengaluru Metropolitan Area, Karnataka**. The BWSSB Annual Report 2020–21 reports an average 1,227 MLD received during that year and a designed treated-water capacity of 1,445 MLD. These city-wide figures provide local context; they are not building-level consumption records and are not used by the prediction model. The source values are included in `data/bengaluru_city_water_context.csv`.

## 2. Environmental Problem

The selected problem is excessive and difficult-to-plan water consumption. When demand is not anticipated, homes and institutions may use water inefficiently and may find it harder to plan conservation.

## 3. Why Does This Problem Matter?

Water is a limited natural resource. Unnecessary consumption and unpredictable demand can lead to wastage. Better prediction can help institutions and communities plan water use. A digital system can use historical data to estimate future demand and support conservation decisions.

## 4. Proposed Digital Solution

Our proposed digital solution is an AI/ML-based web application that uses historical water-consumption and environmental data to predict future water demand and display demand status and conservation recommendations.

React provides the user interface. Spring Boot provides REST APIs and application logic. Spring Data JPA stores and retrieves records. MySQL stores historical consumption and prediction records. The prediction service processes historical records and the new inputs to estimate demand. The estimate is a transparent regression-based ML-style model; it is not deep learning.

## 5. BCS508 Google Form Answers

### Technology Used

- **Artificial Intelligence (AI):** The system applies an automated data-based decision process to estimate demand and suggest conservation actions.
- **Machine Learning (ML):** A simple regression model learns feature weights from saved historical records. It is a small educational model, not a claim of advanced or deep learning.
- **Web Application:** A React website communicates with Spring Boot REST APIs.
- **Data Analytics:** The dashboard calculates and displays consumption summaries and historical trends.

### Why Does This Problem Matter?

Water is a limited natural resource. Unnecessary consumption and unpredictable demand can lead to wastage. Better prediction can help institutions and communities plan water usage. A digital system can use historical data to estimate future demand and support conservation decisions.

### Proposed Digital Solution

An AI/ML-based web application that uses historical water-consumption and environmental data to predict future water demand and display demand status and conservation recommendations. This is our proposed solution. React provides the interface; Spring Boot provides REST APIs and application logic; Spring Data JPA stores and retrieves records; MySQL stores historical consumption and predictions; and prediction logic processes the historical records and supplied inputs to produce predicted demand.

### Input Data Required

- Historical water consumption in litres
- Date, including day of week and month/season
- Temperature
- Rainfall
- Occupancy or population represented by the record

Possible sources include public historical water-consumption datasets, government/open environmental datasets where applicable, weather datasets, and sample/seed records for this prototype. No physical sensor data is claimed or required. The prototype is designed to work without hardware.

### How Does the System Work?

**INPUT** — Historical water use, date, temperature, rainfall, and occupancy.

↓

**PROCESSING** — Validate and store data in MySQL; retrieve historical records through JPA; calculate statistics/features; fit a small regression model; estimate demand; compare it with the normal historical range.

↓

**OUTPUT** — Predicted water demand, NORMAL/HIGH status, historical consumption chart, and a conservation recommendation.

### Expected Environmental Benefits

- Helps users anticipate future water demand.
- Helps identify periods of unusually high expected consumption.
- Supports better water-resource planning.
- Encourages reduction of unnecessary water use.
- Can contribute to reducing water wastage through data-based decisions.

### Developed System / Prototype / Architecture Link

GitHub Repository: https://github.com/shan2302/Waterwise

Local Prototype: Runs locally using React + Spring Boot + MySQL. The final Google Form link can point to the published GitHub repository or another accessible project artifact.

## 6. Technologies Used

- **React and Vite:** Build and serve the browser interface locally.
- **Java 21:** Implements the backend application.
- **Spring Boot and Spring Web:** Run the backend and expose REST endpoints.
- **Spring Data JPA / Hibernate:** Map Java entities to database tables and access records through repositories.
- **MySQL:** Stores records on the local computer in `water_demand_db`.
- **AI/ML:** A small ordinary least-squares regression is fitted from available historical records for prediction.
- **Data Analytics:** Summary statistics and a chart help users understand recorded consumption.

## 7. Input Data

Each water-use record contains its date, consumption in litres, temperature in °C, rainfall in mm, and occupancy count. A prediction request contains a future date and the expected temperature, rainfall, and occupancy. Day of week and month are derived from the date. Records represent a defined building/community scale; sample values are illustrative.

## 8. Possible Data Sources

- **Public datasets:** Public historical water-use records can replace or supplement the demonstration records.
- **Weather/environment datasets:** Public weather and environmental datasets can provide temperature and rainfall inputs.
- **Sample/seed data:** The app inserts clearly illustrative sample records when the database has no usage records. These make the prototype demonstrable without claiming field collection.
- **Published Bengaluru data:** The dashboard presents BWSSB's reported average water received (1,227 MLD) and designed treated-water capacity (1,445 MLD) for the Bengaluru Metropolitan Area. The first is a reported city-wide supply figure for 2020–21; the second is system capacity, not consumption. Both are context only and are excluded from building-level prediction. Source: [BWSSB Annual Report 2020–21, Karnataka Legislative Council](https://kla.kar.nic.in/council/house/Paperlaid/147/91.pdf). The values are also in `data/bengaluru_city_water_context.csv`.

## 9. Input → Processing → Output

```text
INPUT
Historical usage + date + temperature + rainfall + occupancy
        ↓
PROCESSING
Validate → save/retrieve with JPA → analyze history → fit regression → estimate demand
        ↓
OUTPUT
Predicted litres + demand status + chart/statistics + conservation recommendation
```

## 10. System Architecture

```text
React (http://localhost:5173)
        ↓ REST / JSON
Spring Boot REST API (http://localhost:8080)
        ↓
Service Layer
        ↓
Spring Data JPA Repository
        ↓
Local MySQL (water_demand_db)

Historical Data → Prediction Service → Prediction Result → React Dashboard
```

## 11. Database Design

`WaterUsage` stores `id`, `usageDate`, `consumptionLitres`, `temperature`, `rainfall`, and `occupancy`. `Prediction` stores `id`, `predictionDate`, `predictedDemandLitres`, `demandStatus`, `recommendation`, and `createdAt`. Both are ordinary JPA entities. Their repositories extend `JpaRepository`.

## 12. API Endpoints

| Method | URL | Purpose |
|---|---|---|
| GET | `/api/water-usage` | List historical usage records. |
| POST | `/api/water-usage` | Validate and add a usage record. |
| GET | `/api/water-usage/summary` | Return total, average, highest, lowest, and record count. |
| GET | `/api/predictions` | List generated predictions. |
| POST | `/api/predictions` | Generate and save a prediction. |
| GET | `/api/predictions/latest` | Return the newest prediction, or 404 if none exists. |

## 13. Frontend

The dashboard shows total, average, highest, and lowest recorded consumption, the latest prediction and its status, and a historical usage chart. The Water Usage section lists records and provides a form to add one. The Prediction section accepts a date, temperature, rainfall, and occupancy, then displays predicted litres, status, and recommendation.

## 14. Prediction Logic

The service uses saved records to fit an ordinary least-squares linear regression. It uses consumption as the value to estimate and features for temperature, rainfall, occupancy, month, and day of week. The submitted date and environmental values are passed through the fitted model. If there are too few records or the regression cannot be fitted, the service uses a clear historical-average baseline adjusted by occupancy and temperature. The result is kept non-negative.

Demand is HIGH when the estimate is above the historical average by more than 10%; otherwise it is NORMAL. This is a demonstration threshold, not a universal water-safety standard. More and better representative data can improve usefulness. No prediction is hardcoded.

## 15. Environmental Benefits

- Helps users anticipate future water demand.
- Makes unusually high expected demand easier to notice.
- Supports planning for water use.
- Encourages conservation discussions and reduced unnecessary use.
- Supports data-informed efforts to reduce wastage.

## 16. How to Run Locally

The backend needs Java 21, Maven, and a running local MySQL server. The frontend needs Node.js and npm. No paid service, API key, cloud database, hosting, or hardware is used.

## 17. Backend Setup

1. Install Java 21, Maven, and MySQL Server.
2. On Ubuntu, install MySQL Server from a terminal with `sudo apt update` followed by `sudo apt install mysql-server`.
3. Start the service with `sudo systemctl start mysql`. To have it start after computer restarts, run `sudo systemctl enable mysql`.
4. Check that it is running with `sudo systemctl status mysql`. It should show `active (running)`. Press `q` to leave the status view.
5. Open the MySQL console with `sudo mysql` and create the database by entering `CREATE DATABASE water_demand_db;`. Then enter `EXIT;`.
6. The backend uses a local account named `water_app` by default. The username and password are read from `DB_USERNAME` and `DB_PASSWORD`, so the password does not need to be saved in the project. In IntelliJ, set these variables in the run configuration for `WaterDemandApplication`. When running from a terminal, use `DB_USERNAME=water_app DB_PASSWORD='your chosen password' mvn spring-boot:run` from the `backend` folder.
7. In a terminal, go to `backend` and run `mvn spring-boot:run` with those environment variables set, or start `WaterDemandApplication` from IntelliJ.
8. Spring Boot creates the tables and adds sample rows when the usage table is empty.

To create a local MySQL user with a password, enter `sudo mysql`, then run these statements in the MySQL console (replace `choose_a_local_password` with a password you choose):

```sql
CREATE USER 'water_app'@'localhost' IDENTIFIED BY 'choose_a_local_password';
GRANT ALL PRIVILEGES ON water_demand_db.* TO 'water_app'@'localhost';
EXIT;
```

Then provide that same password as the `DB_PASSWORD` environment variable when starting the backend. Do not save your real password in Git or share it in screenshots.

## 18. Frontend Setup

1. Install Node.js (which includes npm).
2. Open another terminal and go to `frontend`.
3. Run `npm install` once.
4. Run `npm run dev`.
5. Open `http://localhost:5173`. The frontend calls the backend at `http://localhost:8080`.

## 19. How to Demonstrate to Faculty

1. Start MySQL.
2. Start Spring Boot from the `backend` folder.
3. Start React from the `frontend` folder.
4. Open `http://localhost:5173`.
5. Show the dashboard and summary cards.
6. Show historical water consumption and its chart.
7. Add a water usage record.
8. Open the prediction form.
9. Enter a prediction date, temperature, rainfall, and occupancy.
10. Generate a prediction.
11. Show the predicted demand in litres.
12. Show the NORMAL/HIGH status.
13. Show the conservation recommendation.
14. Explain the Input → Processing → Output flow.

## 20. Project Limitations

- The included historical records are sample data; prediction quality depends on having enough representative data.
- The published Bengaluru figures describe city-wide utility supply/capacity, not daily consumption at a specific building, so they are not mixed into prediction training.
- The prototype does not directly measure water through physical sensors.
- Prediction accuracy depends on data quality and quantity.
- It does not physically detect leaks or control water supply.
- The simple regression is educational and should not be treated as an operational forecast.

## 21. Future Improvements

- Add larger, reliable real-world datasets.
- Compare with more advanced ML models after obtaining suitable data.
- Optional real-time sensor integration in a separately scoped future version.
- Support records for multiple buildings or areas.
- Improve high-demand/anomaly analysis.
