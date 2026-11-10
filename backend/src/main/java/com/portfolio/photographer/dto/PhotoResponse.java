package com.portfolio.photographer.dto;

public record PhotoResponse(
        Long id,
        String title,
        String description,
        String imageUrl,
        String thumbnailUrl,
        String previewUrl,
        Long categoryId,
        String categoryName,
        boolean featured,
        Integer displayOrder
) {
}
