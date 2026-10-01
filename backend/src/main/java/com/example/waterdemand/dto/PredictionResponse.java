package com.example.waterdemand.dto;

import com.example.waterdemand.entity.Prediction;
import java.time.LocalDate;
import java.time.LocalDateTime;

public record PredictionResponse(Long id, LocalDate predictionDate, double predictedDemandLitres, String demandStatus, String recommendation, LocalDateTime createdAt) {
    public static PredictionResponse from(Prediction prediction) {
        return new PredictionResponse(prediction.getId(), prediction.getPredictionDate(), prediction.getPredictedDemandLitres(), prediction.getDemandStatus(), prediction.getRecommendation(), prediction.getCreatedAt());
    }
}
