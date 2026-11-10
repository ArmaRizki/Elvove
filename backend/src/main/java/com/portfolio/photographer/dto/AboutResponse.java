package com.portfolio.photographer.dto;

public record AboutResponse(
        Long id,
        String introduction,
        Integer yearsOfExperience,
        String photographyStyle,
        String achievements,
        String story,
        String imageUrl
) {
}
