package com.portfolio.photographer.dto;

import jakarta.validation.constraints.AssertTrue;
import jakarta.validation.constraints.Email;
import jakarta.validation.constraints.FutureOrPresent;
import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.NotNull;
import jakarta.validation.constraints.Pattern;

import java.time.LocalDate;
import java.time.LocalTime;

public record BookingRequest(
        @NotBlank(message = "Name is required")
        String name,

        @NotBlank(message = "Email is required")
        @Email(message = "Enter a valid email address")
        String email,

        @NotBlank(message = "Phone is required")
        @Pattern(
                regexp = "^[0-9+\\-\\s]{7,15}$",
                message = "Enter a valid phone number"
        )
        String phone,

        @NotNull(message = "Event date is required")
        @FutureOrPresent(message = "Event date must be today or later")
        LocalDate eventDate,

        @NotNull(message = "Event time is required")
        LocalTime eventTime,

        @NotNull(message = "Package is required")
        Long packageId,

        String eventLocation,
        String budget,
        String message,

        @NotNull(message = "Terms acceptance is required")
        @AssertTrue(message = "You must accept the Terms & Conditions")
        Boolean termsAccepted,

        @NotBlank(message = "Terms version is required")
        String termsVersion
) {
}
