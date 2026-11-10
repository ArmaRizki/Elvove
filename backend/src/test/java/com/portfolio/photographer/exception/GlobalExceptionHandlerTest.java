package com.portfolio.photographer.exception;

import org.junit.jupiter.api.Test;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.multipart.MaxUploadSizeExceededException;
import org.springframework.web.servlet.resource.NoResourceFoundException;

import static org.assertj.core.api.Assertions.assertThat;

/**
 * The upload endpoints answer 413 rather than a generic 500, and the message is
 * the Indonesian one the admin sees in the photo editor.
 */
class GlobalExceptionHandlerTest {

    private final GlobalExceptionHandler handler = new GlobalExceptionHandler();

    @Test
    void oversizeFromTheServiceReturns413() {
        ResponseEntity<ApiError> response =
                handler.handlePayloadTooLarge(new PayloadTooLargeException("Ukuran foto terlalu besar. Maksimal 50 MB."));

        assertThat(response.getStatusCode()).isEqualTo(HttpStatus.PAYLOAD_TOO_LARGE);
        assertThat(response.getBody()).isNotNull();
        assertThat(response.getBody().status()).isEqualTo(413);
        assertThat(response.getBody().message()).contains("50 MB");
    }

    @Test
    void anUnmappedPathReturns404InsteadOfAGeneric500() {
        ResponseEntity<ApiError> response =
                handler.handleNoResource(new NoResourceFoundException(org.springframework.http.HttpMethod.GET, "api/public/galleries"));

        assertThat(response.getStatusCode()).isEqualTo(HttpStatus.NOT_FOUND);
        assertThat(response.getBody()).isNotNull();
        assertThat(response.getBody().status()).isEqualTo(404);
    }

    @Test
    void aMalformedBodyReturns400InsteadOf500() {
        ResponseEntity<ApiError> response = handler.handleUnreadableBody(
                new org.springframework.http.converter.HttpMessageNotReadableException("bad json"));

        assertThat(response.getStatusCode()).isEqualTo(HttpStatus.BAD_REQUEST);
        assertThat(response.getBody()).isNotNull();
        assertThat(response.getBody().status()).isEqualTo(400);
    }

    @Test
    void aDeliberateConflictKeepsItsStatus() {
        ResponseEntity<ApiError> response = handler.handleResponseStatus(
                new org.springframework.web.server.ResponseStatusException(
                        HttpStatus.CONFLICT,
                        "Terms & Conditions have changed. Please review the latest version."));

        assertThat(response.getStatusCode()).isEqualTo(HttpStatus.CONFLICT);
        assertThat(response.getBody()).isNotNull();
        assertThat(response.getBody().status()).isEqualTo(409);
        assertThat(response.getBody().message()).contains("Terms & Conditions have changed");
    }

    @Test
    void oversizeFromTheMultipartParserReturns413() {
        ResponseEntity<ApiError> response =
                handler.handleMaxUploadSize(new MaxUploadSizeExceededException(100L * 1024 * 1024));

        assertThat(response.getStatusCode()).isEqualTo(HttpStatus.PAYLOAD_TOO_LARGE);
        assertThat(response.getBody()).isNotNull();
        assertThat(response.getBody().message()).isEqualTo("Ukuran berkas melebihi batas yang diizinkan.");
    }
}
