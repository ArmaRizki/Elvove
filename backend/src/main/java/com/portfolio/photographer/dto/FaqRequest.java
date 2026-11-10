package com.portfolio.photographer.dto;

import jakarta.validation.constraints.NotBlank;

public record FaqRequest(
        @NotBlank(message = "Question is required") String question,
        @NotBlank(message = "Answer is required") String answer,
        Integer displayOrder
) {
}
