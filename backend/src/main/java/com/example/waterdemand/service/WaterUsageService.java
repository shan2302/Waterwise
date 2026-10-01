package com.example.waterdemand.service;

import com.example.waterdemand.dto.*;
import com.example.waterdemand.entity.WaterUsage;
import com.example.waterdemand.repository.WaterUsageRepository;
import org.springframework.stereotype.Service;
import java.util.Comparator;
import java.util.List;

@Service
public class WaterUsageService {
    private final WaterUsageRepository repository;
    public WaterUsageService(WaterUsageRepository repository) { this.repository = repository; }
    public List<WaterUsageResponse> findAll() {
        return repository.findAll().stream().sorted(Comparator.comparing(WaterUsage::getUsageDate)).map(WaterUsageResponse::from).toList();
    }
    public WaterUsageResponse create(WaterUsageRequest request) {
        WaterUsage saved = repository.save(new WaterUsage(request.usageDate(), request.consumptionLitres(), request.temperature(), request.rainfall(), request.occupancy()));
        return WaterUsageResponse.from(saved);
    }
    public SummaryResponse summary() {
        List<WaterUsage> records = repository.findAll();
        if (records.isEmpty()) return new SummaryResponse(0, 0, 0, 0, 0);
        double total = records.stream().mapToDouble(WaterUsage::getConsumptionLitres).sum();
        double high = records.stream().mapToDouble(WaterUsage::getConsumptionLitres).max().orElse(0);
        double low = records.stream().mapToDouble(WaterUsage::getConsumptionLitres).min().orElse(0);
        return new SummaryResponse(total, total / records.size(), high, low, records.size());
    }
}
