package com.example.waterdemand.controller;

import com.example.waterdemand.dto.*;
import com.example.waterdemand.service.WaterUsageService;
import jakarta.validation.Valid;
import org.springframework.web.bind.annotation.*;
import java.util.List;

@RestController
@RequestMapping("/api/water-usage")
@CrossOrigin(origins = {"http://localhost:5173", "http://127.0.0.1:5173"})
public class WaterUsageController {
    private final WaterUsageService service;
    public WaterUsageController(WaterUsageService service) { this.service = service; }

    /**
     * Returns all historical water consumption records.
     */
    @GetMapping
    public List<WaterUsageResponse> findAll() { return service.findAll(); }

    /**
     * Adds a validated water consumption record.
     */
    @PostMapping
    public WaterUsageResponse create(@Valid @RequestBody WaterUsageRequest request) { return service.create(request); }

    /**
     * Returns summary statistics for historical consumption.
     */
    @GetMapping("/summary")
    public SummaryResponse summary() { return service.summary(); }
}
