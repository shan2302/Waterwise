package com.example.waterdemand.service;

import com.example.waterdemand.dto.PredictionRequest;
import com.example.waterdemand.dto.PredictionResponse;
import com.example.waterdemand.entity.Prediction;
import com.example.waterdemand.entity.WaterUsage;
import com.example.waterdemand.exception.NotFoundException;
import com.example.waterdemand.repository.PredictionRepository;
import com.example.waterdemand.repository.WaterUsageRepository;
import org.springframework.data.domain.Sort;
import org.springframework.stereotype.Service;
import java.util.List;

@Service
public class PredictionService {
    private final WaterUsageRepository usageRepository;
    private final PredictionRepository predictionRepository;
    public PredictionService(WaterUsageRepository usageRepository, PredictionRepository predictionRepository) { this.usageRepository = usageRepository; this.predictionRepository = predictionRepository; }

    public PredictionResponse generate(PredictionRequest request) {
        List<WaterUsage> records = usageRepository.findAll();
        if (records.isEmpty()) throw new IllegalStateException("No historical water usage data is available for prediction.");
        double average = records.stream().mapToDouble(WaterUsage::getConsumptionLitres).average().orElseThrow();
        double estimate = records.size() >= 8 ? regressionEstimate(records, request) : baseline(records, average, request);
        if (!Double.isFinite(estimate) || estimate <= 0) estimate = baseline(records, average, request);
        estimate = Math.max(0, estimate);
        String status = estimate > average * 1.10 ? "HIGH" : "NORMAL";
        String advice = status.equals("HIGH")
                ? "Expected demand is above the normal historical range. Review planned use, check for possible leaks, and reduce non-essential water use."
                : "Demand is within the usual historical range. Continue avoiding unnecessary water use and review consumption regularly.";
        return PredictionResponse.from(predictionRepository.save(new Prediction(request.predictionDate(), Math.round(estimate * 100.0) / 100.0, status, advice)));
    }
    public List<PredictionResponse> findAll() {
        return predictionRepository.findAll(Sort.by(Sort.Direction.DESC, "createdAt")).stream().map(PredictionResponse::from).toList();
    }
    public PredictionResponse latest() {
        return predictionRepository.findAll(Sort.by(Sort.Direction.DESC, "createdAt")).stream().findFirst().map(PredictionResponse::from).orElseThrow(() -> new NotFoundException("No predictions have been generated yet."));
    }

    private double baseline(List<WaterUsage> records, double average, PredictionRequest request) {
        double occupancyAverage = records.stream().mapToInt(WaterUsage::getOccupancy).average().orElse(request.occupancy());
        double temperatureAverage = records.stream().mapToDouble(WaterUsage::getTemperature).average().orElse(request.temperature());
        double occupancyFactor = Math.max(0.5, Math.min(1.8, request.occupancy() / Math.max(1.0, occupancyAverage)));
        double temperatureFactor = Math.max(0.9, Math.min(1.1, 1 + (request.temperature() - temperatureAverage) * 0.005));
        return average * occupancyFactor * temperatureFactor;
    }

    private double regressionEstimate(List<WaterUsage> records, PredictionRequest request) {
        int dimensions = 6;
        double[][] normal = new double[dimensions][dimensions + 1];
        for (WaterUsage record : records) {
            double[] row = features(record.getTemperature(), record.getRainfall(), record.getOccupancy(), record.getUsageDate().getMonthValue(), record.getUsageDate().getDayOfWeek().getValue());
            addRow(normal, row, record.getConsumptionLitres());
        }
        for (int i = 1; i < dimensions; i++) normal[i][i] += 1e-6;
        double[] coefficients = solve(normal);
        double[] input = features(request.temperature(), request.rainfall(), request.occupancy(), request.predictionDate().getMonthValue(), request.predictionDate().getDayOfWeek().getValue());
        double estimate = 0;
        for (int i = 0; i < dimensions; i++) estimate += coefficients[i] * input[i];
        return estimate;
    }
    private double[] features(double temperature, double rainfall, double occupancy, int month, int weekday) {
        return new double[]{1, temperature, rainfall, occupancy, month, weekday};
    }
    private void addRow(double[][] matrix, double[] row, double target) {
        for (int i = 0; i < row.length; i++) {
            for (int j = 0; j < row.length; j++) matrix[i][j] += row[i] * row[j];
            matrix[i][row.length] += row[i] * target;
        }
    }
    private double[] solve(double[][] augmented) {
        int n = augmented.length;
        for (int pivot = 0; pivot < n; pivot++) {
            int best = pivot;
            for (int row = pivot + 1; row < n; row++) if (Math.abs(augmented[row][pivot]) > Math.abs(augmented[best][pivot])) best = row;
            double[] swap = augmented[pivot]; augmented[pivot] = augmented[best]; augmented[best] = swap;
            if (Math.abs(augmented[pivot][pivot]) < 1e-10) throw new IllegalStateException("Historical data could not fit a regression model.");
            double divisor = augmented[pivot][pivot];
            for (int col = pivot; col <= n; col++) augmented[pivot][col] /= divisor;
            for (int row = 0; row < n; row++) if (row != pivot) {
                double factor = augmented[row][pivot];
                for (int col = pivot; col <= n; col++) augmented[row][col] -= factor * augmented[pivot][col];
            }
        }
        double[] result = new double[n];
        for (int row = 0; row < n; row++) result[row] = augmented[row][n];
        return result;
    }
}
