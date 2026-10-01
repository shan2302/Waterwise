package com.example.waterdemand.config;

import com.example.waterdemand.entity.WaterUsage;
import com.example.waterdemand.repository.WaterUsageRepository;
import org.springframework.boot.CommandLineRunner;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import java.time.LocalDate;
import java.util.ArrayList;
import java.util.List;

@Configuration
public class SampleDataConfig {
    @Bean
    CommandLineRunner seedWaterUsage(WaterUsageRepository repository) {
        return args -> {
            if (repository.count() != 0) return;
            List<WaterUsage> samples = new ArrayList<>();
            LocalDate start = LocalDate.now().minusDays(29);
            for (int day = 0; day < 30; day++) {
                LocalDate date = start.plusDays(day);
                double temperature = 25 + (day % 9) * 1.2;
                double rainfall = day % 6 == 0 ? 12 : (day % 3 == 0 ? 3 : 0);
                int occupancy = 86 + (day * 7 % 31);
                double consumption = 4100 + occupancy * 13 + temperature * 38 - rainfall * 16 + (day % 5) * 95;
                samples.add(new WaterUsage(date, Math.round(consumption), temperature, rainfall, occupancy));
            }
            repository.saveAll(samples);
        };
    }
}
