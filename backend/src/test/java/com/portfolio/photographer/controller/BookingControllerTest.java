package com.portfolio.photographer.controller;

import com.portfolio.photographer.exception.TooManyRequestsException;
import com.portfolio.photographer.security.ClientIpResolver;
import com.portfolio.photographer.security.RateLimiterService;
import com.portfolio.photographer.service.BookingService;
import org.junit.jupiter.api.Test;
import org.springframework.http.HttpStatus;
import org.springframework.mock.web.MockHttpServletRequest;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.*;

class BookingControllerTest {

    private MockHttpServletRequest request(String realIp, String xff) {
        MockHttpServletRequest req = new MockHttpServletRequest();
        req.setRemoteAddr("172.18.0.5");
        req.addHeader("X-Real-IP", realIp);
        if (xff != null) req.addHeader("X-Forwarded-For", xff);
        return req;
    }

    @Test
    void bookingLimitKeyedByResolvedIp() {
        BookingService bookingService = mock(BookingService.class);
        RateLimiterService limiter = new RateLimiterService();
        BookingController controller = new BookingController(bookingService, limiter, new ClientIpResolver());

        // 5 allowed per window, even when XFF rotates (XFF is ignored)
        for (int i = 0; i < 5; i++) {
            assertThat(controller.create(null, request("203.0.113.10", "10.0.0." + i)).getStatusCode())
                    .isEqualTo(HttpStatus.CREATED);
        }
        assertThatThrownBy(() -> controller.create(null, request("203.0.113.10", "10.0.0.99")))
                .isInstanceOf(TooManyRequestsException.class);

        // Uses the same key format as before
        assertThat(limiter.isAllowed("booking:203.0.113.10", 5, 600)).isFalse();
        // A different client IP is unaffected
        assertThat(controller.create(null, request("203.0.113.11", null)).getStatusCode())
                .isEqualTo(HttpStatus.CREATED);
        verify(bookingService, times(6)).create(any());
    }
}
