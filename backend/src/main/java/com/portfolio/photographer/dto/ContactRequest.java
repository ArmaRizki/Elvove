package com.portfolio.photographer.dto;

public record ContactRequest(
        String phone,
        String email,
        String address,
        String googleMapLink,
        String whatsappNumber
) {
}
