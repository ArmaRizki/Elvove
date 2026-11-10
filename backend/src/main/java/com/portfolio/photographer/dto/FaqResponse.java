package com.portfolio.photographer.dto;

public record FaqResponse(
        Long id,
        String question,
        String answer,
        Integer displayOrder
) {
}
