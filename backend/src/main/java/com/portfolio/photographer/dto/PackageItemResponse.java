package com.portfolio.photographer.dto;

import java.math.BigDecimal;
import java.util.List;

public record PackageItemResponse(
        Long id,
        String title,
        BigDecimal price,
        List<String> features,
        Integer displayOrder,
        boolean highlighted
) {
}
