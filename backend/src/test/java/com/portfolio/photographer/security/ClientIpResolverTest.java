package com.portfolio.photographer.security;

import org.junit.jupiter.api.Test;
import org.springframework.mock.web.MockHttpServletRequest;

import static org.assertj.core.api.Assertions.assertThat;

class ClientIpResolverTest {

    private final ClientIpResolver resolver = new ClientIpResolver();

    @Test
    void usesXRealIp() {
        MockHttpServletRequest req = new MockHttpServletRequest();
        req.setRemoteAddr("172.18.0.5");
        req.addHeader("X-Real-IP", " 203.0.113.7 ");
        assertThat(resolver.resolve(req)).isEqualTo("203.0.113.7");
    }

    @Test
    void ignoresXForwardedFor() {
        MockHttpServletRequest req = new MockHttpServletRequest();
        req.setRemoteAddr("172.18.0.5");
        req.addHeader("X-Forwarded-For", "1.1.1.1, 203.0.113.7");
        assertThat(resolver.resolve(req)).isEqualTo("172.18.0.5");

        req.addHeader("X-Real-IP", "203.0.113.7");
        assertThat(resolver.resolve(req)).isEqualTo("203.0.113.7");
    }

    @Test
    void fallsBackToRemoteAddrWhenBlank() {
        MockHttpServletRequest req = new MockHttpServletRequest();
        req.setRemoteAddr("10.0.0.9");
        req.addHeader("X-Real-IP", "   ");
        assertThat(resolver.resolve(req)).isEqualTo("10.0.0.9");
    }

    @Test
    void capsLength() {
        MockHttpServletRequest req = new MockHttpServletRequest();
        req.addHeader("X-Real-IP", "x".repeat(5000));
        assertThat(resolver.resolve(req)).hasSize(ClientIpResolver.MAX_LENGTH);
    }
}
