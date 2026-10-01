package com.example.waterdemand.controller;

import com.example.waterdemand.dto.PredictionRequest;
import com.example.waterdemand.dto.PredictionResponse;
import com.example.waterdemand.service.PredictionService;
import jakarta.validation.Valid;
import org.springframework.web.bind.annotation.*;
import java.util.List;

@RestController
@RequestMapping("/api/predictions")
@CrossOrigin(origins = {"http://localhost:5173", "http://127.0.0.1:5173"})
public class PredictionController {
    private final PredictionService service;
    public PredictionController(PredictionService service) { this.service = service; }

    /**
     * Returns previously generated water demand predictions.
     */
    @GetMapping
    public List<PredictionResponse> findAll() { return service.findAll(); }

    /**
     * Generates a water demand prediction from supplied inputs and historical records.
     */
    @PostMapping
    public PredictionResponse generate(@Valid @RequestBody PredictionRequest request) { return service.generate(request); }

    /**
     * Returns the most recently generated water demand prediction.
     */
    @GetMapping("/latest")
    public PredictionResponse latest() { return service.latest(); }
}
