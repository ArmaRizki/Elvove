package com.portfolio.photographer.dto;

import jakarta.validation.constraints.Min;

public record AboutRequest(
        String introduction,
        @Min(value = 0, message = "Years of experience cannot be negative") Integer yearsOfExperience,
        String photographyStyle,
        String achievements,
        String story,
        String imageUrl
) {
}
