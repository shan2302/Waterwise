package com.example.waterdemand.dto;

import jakarta.validation.constraints.*;
import java.time.LocalDate;

public record PredictionRequest(@NotNull LocalDate predictionDate,
                                @NotNull @DecimalMin("-20.0") @DecimalMax("60.0") Double temperature,
                                @NotNull @DecimalMin("0.0") @DecimalMax("1000.0") Double rainfall,
                                @NotNull @Min(1) @Max(1000000) Integer occupancy) {}
