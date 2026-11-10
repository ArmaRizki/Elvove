package com.portfolio.photographer.controller;

import jakarta.validation.Validation;
import jakarta.validation.Validator;
import jakarta.validation.ValidatorFactory;
import org.junit.jupiter.api.AfterAll;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Test;

import java.util.List;

import static org.assertj.core.api.Assertions.assertThat;

/**
 * Request-shape rules for the booking-notification settings endpoint. A missing
 * field must be a 400 (handled by the validation advice), never a 500, and an
 * absent {@code recipients} field must not be read as "clear the list".
 */
class BookingNotificationSettingsControllerTest {

    private static ValidatorFactory factory;
    private static Validator validator;

    @BeforeAll
    static void setUpValidator() {
        factory = Validation.buildDefaultValidatorFactory();
        validator = factory.getValidator();
    }

    @AfterAll
    static void closeValidator() {
        factory.close();
    }

    @Test
    void missingRecipientsIsRejected() {
        var request = new BookingNotificationSettingsController.SaveRequest(true, null);

        assertThat(validator.validate(request))
                .anyMatch(v -> v.getPropertyPath().toString().equals("recipients"));
    }

    @Test
    void missingEnabledIsRejected() {
        var request = new BookingNotificationSettingsController.SaveRequest(null, List.of("a@contoh.com"));

        assertThat(validator.validate(request))
                .anyMatch(v -> v.getPropertyPath().toString().equals("enabled"));
    }

    @Test
    void emptyRecipientArrayIsValidSoTheListCanBeCleared() {
        var request = new BookingNotificationSettingsController.SaveRequest(false, List.of());

        assertThat(validator.validate(request)).isEmpty();
    }

    @Test
    void aPopulatedListIsValid() {
        var request = new BookingNotificationSettingsController.SaveRequest(
                true, List.of("a@contoh.com", "b@contoh.com"));

        assertThat(validator.validate(request)).isEmpty();
    }
}
