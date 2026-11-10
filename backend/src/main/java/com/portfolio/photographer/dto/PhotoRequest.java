package com.portfolio.photographer.dto;

import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.NotNull;

public record PhotoRequest(
        @NotBlank(message = "Title is required") String title,
        String description,
        @NotBlank(message = "Image URL is required") String imageUrl,
        @NotNull(message = "Category is required") Long categoryId,
        boolean featured,
        Integer displayOrder
) {
}
