package com.example.waterdemand.entity;

import jakarta.persistence.*;
import java.time.LocalDate;
import java.time.LocalDateTime;

@Entity
@Table(name = "predictions")
public class Prediction {
    @Id @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Long id;
    @Column(nullable = false)
    private LocalDate predictionDate;
    @Column(nullable = false)
    private double predictedDemandLitres;
    @Column(nullable = false)
    private String demandStatus;
    @Column(nullable = false, length = 500)
    private String recommendation;
    @Column(nullable = false)
    private LocalDateTime createdAt;

    protected Prediction() {}
    public Prediction(LocalDate predictionDate, double predictedDemandLitres, String demandStatus, String recommendation) {
        this.predictionDate = predictionDate; this.predictedDemandLitres = predictedDemandLitres; this.demandStatus = demandStatus; this.recommendation = recommendation; this.createdAt = LocalDateTime.now();
    }
    public Long getId() { return id; }
    public LocalDate getPredictionDate() { return predictionDate; }
    public double getPredictedDemandLitres() { return predictedDemandLitres; }
    public String getDemandStatus() { return demandStatus; }
    public String getRecommendation() { return recommendation; }
    public LocalDateTime getCreatedAt() { return createdAt; }
}
