package com.example.waterdemand.service;

import com.example.waterdemand.dto.PredictionRequest;
import com.example.waterdemand.entity.WaterUsage;
import com.example.waterdemand.repository.PredictionRepository;
import com.example.waterdemand.repository.WaterUsageRepository;
import org.junit.jupiter.api.Test;
import java.time.LocalDate;
import java.util.ArrayList;
import java.util.List;
import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.*;

class PredictionServiceTest {
    @Test
    void predictsFromHistoryAndPersistsResult() {
        WaterUsageRepository usages = mock(WaterUsageRepository.class);
        PredictionRepository predictions = mock(PredictionRepository.class);
        List<WaterUsage> history = new ArrayList<>();
        for (int day = 0; day < 14; day++) history.add(new WaterUsage(LocalDate.of(2026, 9, 1).plusDays(day), 6000 + day * 80, 26 + day % 5, day % 4, 90 + day));
        when(usages.findAll()).thenReturn(history);
        when(predictions.save(any())).thenAnswer(invocation -> invocation.getArgument(0));
        PredictionService service = new PredictionService(usages, predictions);

        var response = service.generate(new PredictionRequest(LocalDate.of(2026, 10, 3), 31.0, 0.0, 150));

        assertTrue(response.predictedDemandLitres() > 0);
        assertTrue(List.of("NORMAL", "HIGH").contains(response.demandStatus()));
        assertNotNull(response.recommendation());
        verify(predictions).save(any());
    }
}
