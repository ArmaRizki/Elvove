package com.portfolio.photographer.exception;

import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.http.converter.HttpMessageNotReadableException;
import org.springframework.security.authentication.BadCredentialsException;
import org.springframework.validation.FieldError;
import org.springframework.web.bind.MethodArgumentNotValidException;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.RestControllerAdvice;
import org.springframework.web.method.annotation.MethodArgumentTypeMismatchException;
import org.springframework.web.multipart.MaxUploadSizeExceededException;
import org.springframework.web.server.ResponseStatusException;
import org.springframework.web.servlet.resource.NoResourceFoundException;
import lombok.extern.slf4j.Slf4j;

import java.time.LocalDateTime;
import java.util.HashMap;
import java.util.Map;

@Slf4j
@RestControllerAdvice
public class GlobalExceptionHandler {

    @ExceptionHandler(ResourceNotFoundException.class)
    public ResponseEntity<ApiError> handleNotFound(ResourceNotFoundException ex) {
        ApiError error = new ApiError(
                LocalDateTime.now(), HttpStatus.NOT_FOUND.value(),
                "Not Found", ex.getMessage(), null
        );
        return ResponseEntity.status(HttpStatus.NOT_FOUND).body(error);
    }

    @ExceptionHandler(BadRequestException.class)
    public ResponseEntity<ApiError> handleBadRequest(BadRequestException ex) {
        ApiError error = new ApiError(
                LocalDateTime.now(), HttpStatus.BAD_REQUEST.value(),
                "Bad Request", ex.getMessage(), null
        );
        return ResponseEntity.status(HttpStatus.BAD_REQUEST).body(error);
    }

    @ExceptionHandler(TooManyRequestsException.class)
    public ResponseEntity<ApiError> handleTooManyRequests(TooManyRequestsException ex) {
        ApiError error = new ApiError(
                LocalDateTime.now(), HttpStatus.TOO_MANY_REQUESTS.value(),
                "Too Many Requests", ex.getMessage(), null
        );
        return ResponseEntity.status(HttpStatus.TOO_MANY_REQUESTS).body(error);
    }

    // Thrown by FileStorageService when the file itself is over the configured
    // limit (a single file larger than max-file-size, or a limit reached before
    // the multipart parser sees it).
    @ExceptionHandler(PayloadTooLargeException.class)
    public ResponseEntity<ApiError> handlePayloadTooLarge(PayloadTooLargeException ex) {
        ApiError error = new ApiError(
                LocalDateTime.now(), HttpStatus.PAYLOAD_TOO_LARGE.value(),
                "Payload Too Large", ex.getMessage(), null
        );
        return ResponseEntity.status(HttpStatus.PAYLOAD_TOO_LARGE).body(error);
    }

    // Thrown by the multipart parser before the controller runs: the upload is
    // larger than spring.servlet.multipart.max-file-size / max-request-size.
    @ExceptionHandler(MaxUploadSizeExceededException.class)
    public ResponseEntity<ApiError> handleMaxUploadSize(MaxUploadSizeExceededException ex) {
        ApiError error = new ApiError(
                LocalDateTime.now(), HttpStatus.PAYLOAD_TOO_LARGE.value(),
                "Payload Too Large",
                "Ukuran berkas melebihi batas yang diizinkan.", null
        );
        return ResponseEntity.status(HttpStatus.PAYLOAD_TOO_LARGE).body(error);
    }

    @ExceptionHandler(BadCredentialsException.class)
    public ResponseEntity<ApiError> handleBadCredentials(BadCredentialsException ex) {
        ApiError error = new ApiError(
                LocalDateTime.now(), HttpStatus.UNAUTHORIZED.value(),
                "Unauthorized", "Invalid username or password", null
        );
        return ResponseEntity.status(HttpStatus.UNAUTHORIZED).body(error);
    }

    @ExceptionHandler(MethodArgumentNotValidException.class)
    public ResponseEntity<ApiError> handleValidation(MethodArgumentNotValidException ex) {
        Map<String, String> fieldErrors = new HashMap<>();
        for (FieldError fe : ex.getBindingResult().getFieldErrors()) {
            fieldErrors.put(fe.getField(), fe.getDefaultMessage());
        }
        ApiError error = new ApiError(
                LocalDateTime.now(), HttpStatus.BAD_REQUEST.value(),
                "Validation Failed", "One or more fields are invalid", fieldErrors
        );
        return ResponseEntity.status(HttpStatus.BAD_REQUEST).body(error);
    }

    // A body the client cannot parse (truncated JSON, wrong content type) is a
    // client error, not a server failure.
    @ExceptionHandler({HttpMessageNotReadableException.class, MethodArgumentTypeMismatchException.class})
    public ResponseEntity<ApiError> handleUnreadableBody(Exception ex) {
        ApiError error = new ApiError(
                LocalDateTime.now(), HttpStatus.BAD_REQUEST.value(),
                "Bad Request", "Format permintaan tidak valid. Periksa kembali data yang dikirim.", null
        );
        return ResponseEntity.status(HttpStatus.BAD_REQUEST).body(error);
    }

    @ExceptionHandler(NoResourceFoundException.class)
    public ResponseEntity<ApiError> handleNoResource(NoResourceFoundException ex) {
        ApiError error = new ApiError(
                LocalDateTime.now(), HttpStatus.NOT_FOUND.value(),
                "Not Found", "Halaman atau sumber daya tidak ditemukan.", null
        );
        return ResponseEntity.status(HttpStatus.NOT_FOUND).body(error);
    }

    // Services raise ResponseStatusException for deliberate status codes (for
    // example 409 when the terms version changed). Without this handler the
    // catch-all below would turn them into a 500 and the client could no longer
    // tell the two apart.
    @ExceptionHandler(ResponseStatusException.class)
    public ResponseEntity<ApiError> handleResponseStatus(ResponseStatusException ex) {
        HttpStatus status = HttpStatus.resolve(ex.getStatusCode().value());
        if (status == null) status = HttpStatus.INTERNAL_SERVER_ERROR;
        String message = ex.getReason() != null ? ex.getReason() : status.getReasonPhrase();
        if (status.is5xxServerError()) log.error("Request failed with {}", status, ex);
        ApiError error = new ApiError(
                LocalDateTime.now(), status.value(), status.getReasonPhrase(), message, null
        );
        return ResponseEntity.status(status).body(error);
    }

    @ExceptionHandler(Exception.class)
    public ResponseEntity<ApiError> handleGeneric(Exception ex) {
        log.error("Unhandled exception", ex);
        ApiError error = new ApiError(
                LocalDateTime.now(), HttpStatus.INTERNAL_SERVER_ERROR.value(),
                "Internal Server Error", "Something went wrong. Please try again.", null
        );
        return ResponseEntity.status(HttpStatus.INTERNAL_SERVER_ERROR).body(error);
    }
}
