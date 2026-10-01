package com.example.waterdemand.service;

import com.example.waterdemand.dto.WaterUsageRequest;
import com.example.waterdemand.entity.WaterUsage;
import com.example.waterdemand.repository.WaterUsageRepository;
import org.junit.jupiter.api.Test;
import java.time.LocalDate;
import java.util.List;
import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.Mockito.*;

class WaterUsageServiceTest {
    @Test
    void createsAndRetrievesUsageAndCalculatesSummary() {
        WaterUsageRepository repository = mock(WaterUsageRepository.class);
        WaterUsage saved = new WaterUsage(LocalDate.of(2026, 10, 1), 5000, 28, 2, 100);
        when(repository.save(any(WaterUsage.class))).thenReturn(saved);
        when(repository.findAll()).thenReturn(List.of(saved, new WaterUsage(LocalDate.of(2026, 9, 30), 7000, 30, 0, 110)));
        WaterUsageService service = new WaterUsageService(repository);

        assertEquals(5000, service.create(new WaterUsageRequest(LocalDate.of(2026, 10, 1), 5000.0, 28.0, 2.0, 100)).consumptionLitres());
        assertEquals(2, service.findAll().size());
        assertEquals(6000, service.summary().averageConsumption());
        assertEquals(12000, service.summary().totalConsumption());
    }
}
