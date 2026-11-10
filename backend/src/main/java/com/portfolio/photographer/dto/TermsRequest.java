package com.portfolio.photographer.dto;

import jakarta.validation.constraints.NotBlank;

public record TermsRequest(
        @NotBlank String title,
        String introduction,
        @NotBlank String content
) {}