package com.example.waterdemand.repository;

import com.example.waterdemand.entity.WaterUsage;
import org.springframework.data.jpa.repository.JpaRepository;

public interface WaterUsageRepository extends JpaRepository<WaterUsage, Long> {}
