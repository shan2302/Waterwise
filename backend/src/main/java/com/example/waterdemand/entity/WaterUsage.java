package com.example.waterdemand.entity;

import jakarta.persistence.*;
import java.time.LocalDate;

@Entity
@Table(name = "water_usage")
public class WaterUsage {
    @Id @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Long id;
    @Column(nullable = false)
    private LocalDate usageDate;
    @Column(nullable = false)
    private double consumptionLitres;
    @Column(nullable = false)
    private double temperature;
    @Column(nullable = false)
    private double rainfall;
    @Column(nullable = false)
    private int occupancy;

    protected WaterUsage() {}
    public WaterUsage(LocalDate usageDate, double consumptionLitres, double temperature, double rainfall, int occupancy) {
        this.usageDate = usageDate; this.consumptionLitres = consumptionLitres; this.temperature = temperature; this.rainfall = rainfall; this.occupancy = occupancy;
    }
    public Long getId() { return id; }
    public LocalDate getUsageDate() { return usageDate; }
    public double getConsumptionLitres() { return consumptionLitres; }
    public double getTemperature() { return temperature; }
    public double getRainfall() { return rainfall; }
    public int getOccupancy() { return occupancy; }
}
