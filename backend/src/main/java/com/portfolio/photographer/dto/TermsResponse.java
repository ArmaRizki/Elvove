package com.portfolio.photographer.dto;

import java.time.LocalDateTime;

public record TermsResponse(
        Long id,
        String title,
        String introduction,
        String content,
        String version,
        LocalDateTime updatedAt
) {}