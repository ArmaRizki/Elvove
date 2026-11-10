package com.portfolio.photographer.dto;

public record LoginResponse(
        String token,
        String username,
        String email
) {
}
