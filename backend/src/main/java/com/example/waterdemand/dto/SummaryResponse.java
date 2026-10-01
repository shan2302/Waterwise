package com.example.waterdemand.dto;

public record SummaryResponse(double totalConsumption, double averageConsumption, double highestConsumption, double lowestConsumption, long recordCount) {}
