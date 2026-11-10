package com.portfolio.photographer.security;

import org.junit.jupiter.api.Test;
import org.springframework.security.core.userdetails.User;
import org.springframework.security.core.userdetails.UserDetails;

import java.util.Collections;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

class JwtServiceTest {

    private static final String SECRET_64 = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef";

    @Test
    void blankSecretFails() {
        assertThatThrownBy(() -> new JwtService("", 60_000))
                .isInstanceOf(IllegalStateException.class).hasMessageContaining("JWT_SECRET");
        assertThatThrownBy(() -> new JwtService("   ", 60_000))
                .isInstanceOf(IllegalStateException.class);
        assertThatThrownBy(() -> new JwtService(null, 60_000))
                .isInstanceOf(IllegalStateException.class);
    }

    @Test
    void shortSecretFails() {
        String thirtyOne = "a".repeat(31);
        assertThatThrownBy(() -> new JwtService(thirtyOne, 60_000))
                .isInstanceOf(IllegalStateException.class)
                .hasMessage("JWT_SECRET must be at least 32 bytes");
    }

    @Test
    void validSecretRoundTrip() {
        JwtService service = new JwtService(SECRET_64, 60_000);
        UserDetails user = new User("owner", "x", Collections.emptyList());
        UserDetails other = new User("someone-else", "x", Collections.emptyList());

        String token = service.generateToken(user);

        assertThat(service.extractUsername(token)).isEqualTo("owner");
        assertThat(service.isTokenValid(token, user)).isTrue();
        assertThat(service.isTokenValid(token, other)).isFalse();
    }

    @Test
    void tokenFromSameSecretIsValidAcrossInstances() {
        // Same secret => existing tokens stay valid after a restart
        String token = new JwtService(SECRET_64, 60_000)
                .generateToken(new User("owner", "x", Collections.emptyList()));
        assertThat(new JwtService(SECRET_64, 60_000).extractUsername(token)).isEqualTo("owner");
    }
}
