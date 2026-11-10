package com.portfolio.photographer.controller;

import com.portfolio.photographer.dto.LoginRequest;
import com.portfolio.photographer.dto.LoginResponse;
import com.portfolio.photographer.exception.GlobalExceptionHandler;
import com.portfolio.photographer.security.ClientIpResolver;
import com.portfolio.photographer.security.RateLimiterService;
import com.portfolio.photographer.service.AuthService;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.http.MediaType;
import org.springframework.security.authentication.BadCredentialsException;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.ResultActions;
import org.springframework.test.web.servlet.setup.MockMvcBuilders;

import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.argThat;
import static org.mockito.Mockito.*;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

class AuthControllerTest {

    private AuthService authService;
    private MockMvc mockMvc;

    @BeforeEach
    void setUp() {
        authService = mock(AuthService.class);
        AuthController controller = new AuthController(authService, new RateLimiterService(), new ClientIpResolver());
        mockMvc = MockMvcBuilders.standaloneSetup(controller)
                .setControllerAdvice(new GlobalExceptionHandler())
                .build();
    }

    private ResultActions login(String username, String password, String realIp, String xff) throws Exception {
        var req = post("/api/auth/login")
                .contentType(MediaType.APPLICATION_JSON)
                .content("{\"username\":\"" + username + "\",\"password\":\"" + password + "\"}")
                .header("X-Real-IP", realIp);
        if (xff != null) {
            req.header("X-Forwarded-For", xff);
        }
        return mockMvc.perform(req);
    }

    private void failFor(String password) {
        when(authService.login(argThat(r -> r != null && password.equals(r.password()))))
                .thenThrow(new BadCredentialsException("Bad credentials"));
    }

    private void succeedFor(String password) {
        when(authService.login(argThat(r -> r != null && password.equals(r.password()))))
                .thenReturn(new LoginResponse("token", "owner", "owner@example.com"));
    }

    @Test
    void fiveFailuresThenLockedEvenWithCorrectPassword() throws Exception {
        failFor("wrong");
        succeedFor("right");

        for (int i = 0; i < 5; i++) {
            login("owner", "wrong", "198.51.100.1", null)
                    .andExpect(status().isUnauthorized())
                    .andExpect(jsonPath("$.message").value("Invalid username or password"));
        }

        login("owner", "right", "198.51.100.1", null)
                .andExpect(status().isTooManyRequests())
                .andExpect(jsonPath("$.status").value(429))
                .andExpect(jsonPath("$.error").value("Too Many Requests"))
                .andExpect(jsonPath("$.message").value("Too many login attempts. Please try again later."))
                .andExpect(jsonPath("$.timestamp").exists());

        // 5 failing calls reached AuthService; the locked-out attempt did not
        verify(authService, times(5)).login(any(LoginRequest.class));
    }

    @Test
    void usernameKeyIsCaseAndWhitespaceInsensitive() throws Exception {
        failFor("wrong");
        String[] variants = {"Owner", "OWNER", " owner ", "owner", "oWnEr"};
        for (String v : variants) {
            login(v, "wrong", "198.51.100.2", null).andExpect(status().isUnauthorized());
        }
        login("owner", "wrong", "198.51.100.2", null).andExpect(status().isTooManyRequests());
    }

    @Test
    void successBeforeLimitResetsCounter() throws Exception {
        failFor("wrong");
        succeedFor("right");

        for (int i = 0; i < 4; i++) {
            login("owner", "wrong", "198.51.100.3", null).andExpect(status().isUnauthorized());
        }
        login("owner", "right", "198.51.100.3", null)
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.token").value("token"));

        // Counter was reset: another 4 failures are still 401, not 429
        for (int i = 0; i < 4; i++) {
            login("owner", "wrong", "198.51.100.3", null).andExpect(status().isUnauthorized());
        }
    }

    @Test
    void lockoutFromOneIpDoesNotAffectAnotherIp() throws Exception {
        failFor("wrong");
        succeedFor("right");

        // (a) 5 failures for "admin" from 203.0.113.10 -> 6th from that IP is 429, even with the right password
        for (int i = 0; i < 5; i++) {
            login("admin", "wrong", "203.0.113.10", null).andExpect(status().isUnauthorized());
        }
        login("admin", "right", "203.0.113.10", null)
                .andExpect(status().isTooManyRequests())
                .andExpect(jsonPath("$.status").value(429))
                .andExpect(jsonPath("$.message").value("Too many login attempts. Please try again later."));
        verify(authService, times(5)).login(any(LoginRequest.class));

        // (b) Same username from 203.0.113.20 still reaches AuthService
        login("admin", "wrong", "203.0.113.20", null)
                .andExpect(status().isUnauthorized())
                .andExpect(jsonPath("$.message").value("Invalid username or password"));
        login("admin", "right", "203.0.113.20", null)
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.token").value("token"));
        verify(authService, times(7)).login(any(LoginRequest.class));

        // The original IP is still locked out
        login("admin", "right", "203.0.113.10", null).andExpect(status().isTooManyRequests());
    }

    @Test
    void rotatingXffWithSameRealIpDoesNotBypassLockout() throws Exception {
        failFor("wrong");
        succeedFor("right");

        // (c) X-Forwarded-For is ignored, so rotating it keeps the same username+IP key
        for (int i = 0; i < 5; i++) {
            login("owner", "wrong", "203.0.113.30", "10.9.9." + i).andExpect(status().isUnauthorized());
        }
        login("owner", "right", "203.0.113.30", "1.2.3.4").andExpect(status().isTooManyRequests());
    }

    @Test
    void successResetsOnlyThatUsernameAndIpCounter() throws Exception {
        failFor("wrong");
        succeedFor("right");

        // (e) 4 failures for "owner" from two IPs
        for (int i = 0; i < 4; i++) {
            login("owner", "wrong", "203.0.113.40", null).andExpect(status().isUnauthorized());
            login("owner", "wrong", "203.0.113.41", null).andExpect(status().isUnauthorized());
        }
        // Success from .40 resets only the .40 counter
        login("owner", "right", "203.0.113.40", null).andExpect(status().isOk());

        // .41 still has 4 failures: one more failure reaches the limit, then 429
        login("owner", "wrong", "203.0.113.41", null).andExpect(status().isUnauthorized());
        login("owner", "right", "203.0.113.41", null).andExpect(status().isTooManyRequests());

        // .40 was reset: 4 more failures are still 401
        for (int i = 0; i < 4; i++) {
            login("owner", "wrong", "203.0.113.40", null).andExpect(status().isUnauthorized());
        }
        login("owner", "right", "203.0.113.40", null).andExpect(status().isOk());
    }

    @Test
    void spoofedXffDoesNotChangeIpKey() throws Exception {
        failFor("wrong");
        // Same real IP, different XFF and usernames: all count against one IP bucket
        for (int i = 0; i < 20; i++) {
            login("user" + i, "wrong", "198.51.100.4", "10.0.0." + i).andExpect(status().isUnauthorized());
        }
        login("another", "wrong", "198.51.100.4", "10.0.0.250").andExpect(status().isTooManyRequests());
    }

    @Test
    void perIpLimitAfterTwentyAttemptsAcrossUsernames() throws Exception {
        failFor("wrong");
        succeedFor("right");

        for (int i = 0; i < 20; i++) {
            login("user" + i, "wrong", "198.51.100.5", null).andExpect(status().isUnauthorized());
        }
        login("fresh-user", "right", "198.51.100.5", null)
                .andExpect(status().isTooManyRequests())
                .andExpect(jsonPath("$.status").value(429));

        // A different IP is unaffected
        login("fresh-user", "right", "198.51.100.6", null).andExpect(status().isOk());
    }

    @Test
    void veryLongUsernameIsStillLockedOut() throws Exception {
        failFor("wrong");
        succeedFor("right");
        String longUser = "u".repeat(10_000);

        for (int i = 0; i < 5; i++) {
            login(longUser, "wrong", "198.51.100.8", null).andExpect(status().isUnauthorized());
        }
        // Same username, different casing/whitespace normalizes to the same hashed key
        login("  " + longUser.toUpperCase() + " ", "right", "198.51.100.8", null)
                .andExpect(status().isTooManyRequests())
                .andExpect(jsonPath("$.status").value(429));
    }

    @Test
    void distinctLongUsernamesDoNotShareCounter() throws Exception {
        failFor("wrong");
        succeedFor("right");
        String longA = "a".repeat(5_000) + "-one";
        String longB = "a".repeat(5_000) + "-two";

        for (int i = 0; i < 5; i++) {
            login(longA, "wrong", "198.51.100.9", null).andExpect(status().isUnauthorized());
        }
        login(longA, "right", "198.51.100.9", null).andExpect(status().isTooManyRequests());

        // Same long prefix, different username: separate counter, not locked
        login(longB, "wrong", "198.51.100.9", null).andExpect(status().isUnauthorized());
        login(longB, "right", "198.51.100.9", null).andExpect(status().isOk());
    }

    @Test
    void blankUsernameStillRejectedByValidation() throws Exception {
        login("", "x", "198.51.100.7", null).andExpect(status().isBadRequest());
        verifyNoInteractions(authService);
    }
}
