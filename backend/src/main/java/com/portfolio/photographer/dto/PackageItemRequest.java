package com.portfolio.photographer.dto;

import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.NotNull;
import jakarta.validation.constraints.Positive;

import java.math.BigDecimal;
import java.util.List;

public record PackageItemRequest(
        @NotBlank(message = "Title is required") String title,
        @NotNull(message = "Price is required") @Positive(message = "Price must be positive") BigDecimal price,
        List<String> features,
        Integer displayOrder,
        boolean highlighted
) {
}
