package com.example.waterdemand.dto;

import com.example.waterdemand.entity.WaterUsage;
import java.time.LocalDate;

public record WaterUsageResponse(Long id, LocalDate usageDate, double consumptionLitres, double temperature, double rainfall, int occupancy) {
    public static WaterUsageResponse from(WaterUsage usage) {
        return new WaterUsageResponse(usage.getId(), usage.getUsageDate(), usage.getConsumptionLitres(), usage.getTemperature(), usage.getRainfall(), usage.getOccupancy());
    }
}
