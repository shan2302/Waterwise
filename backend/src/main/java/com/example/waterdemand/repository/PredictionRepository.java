package com.example.waterdemand.repository;

import com.example.waterdemand.entity.Prediction;
import org.springframework.data.jpa.repository.JpaRepository;

public interface PredictionRepository extends JpaRepository<Prediction, Long> {}
