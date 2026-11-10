package com.portfolio.photographer.dto;

public record SocialLinksResponse(
        Long id,
        String instagram,
        String threads,
        String tiktok,
        String facebook,
        String youtube,
        String pinterest,
        String linkedin
) {
}
