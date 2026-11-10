package com.portfolio.photographer.dto;

public record ContactResponse(
        Long id,
        String phone,
        String email,
        String address,
        String googleMapLink,
        String whatsappNumber
) {
}
