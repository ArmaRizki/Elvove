package com.portfolio.photographer.exception;

// An upload that is valid in every other way but larger than the configured
// limit. Kept apart from BadRequestException so it answers 413, the same status
// the multipart parser uses for an oversized request.
public class PayloadTooLargeException extends RuntimeException {
    public PayloadTooLargeException(String message) {
        super(message);
    }
}
