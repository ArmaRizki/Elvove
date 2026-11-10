package com.portfolio.photographer.controller;

import com.portfolio.photographer.dto.LoginRequest;
import com.portfolio.photographer.dto.LoginResponse;
import com.portfolio.photographer.exception.TooManyRequestsException;
import com.portfolio.photographer.security.ClientIpResolver;
import com.portfolio.photographer.security.RateLimiterService;
import com.portfolio.photographer.service.AuthService;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.validation.Valid;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.security.core.AuthenticationException;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.util.HexFormat;
import java.util.Locale;

/**
 * Login endpoint with two in-memory rate limits:
 * <ul>
 *   <li>Per client IP ({@code login-ip:<ip>}): every attempt counts, max 20 per 15 minutes.</li>
 *   <li>Per username + client IP ({@code login-user:<ip>|<username>}): only failures count,
 *       max 5 per 15 minutes. Keying on the IP too means an attacker on another IP
 *       cannot lock the real admin out.</li>
 * </ul>
 * The IP comes from {@link ClientIpResolver} (X-Real-IP, capped at 64 chars); the username
 * part is trimmed and lowercased, or its SHA-256 hex when longer than 128 chars.
 */
@RestController
@RequestMapping("/api/auth")
@RequiredArgsConstructor
public class AuthController {

    private static final int MAX_ATTEMPTS_PER_IP = 20;
    private static final int MAX_FAILURES_PER_USER = 5;
    private static final long WINDOW_SECONDS = 900; // 15 minutes
    private static final int MAX_USERNAME_KEY_LENGTH = 128;
    private static final String TOO_MANY_ATTEMPTS = "Too many login attempts. Please try again later.";

    private final AuthService authService;
    private final RateLimiterService rateLimiterService;
    private final ClientIpResolver clientIpResolver;

    @PostMapping("/login")
    public ResponseEntity<LoginResponse> login(@Valid @RequestBody LoginRequest request, HttpServletRequest httpRequest) {
        // Per IP: every attempt counts
        String ip = clientIpResolver.resolve(httpRequest);
        if (!rateLimiterService.isAllowed("login-ip:" + ip, MAX_ATTEMPTS_PER_IP, WINDOW_SECONDS)) {
            throw new TooManyRequestsException(TOO_MANY_ATTEMPTS);
        }

        // Per username + IP ("login-user:<ip>|<username>"): only failures count; a locked-out
        // username is blocked from this IP even with the right password, other IPs are unaffected
        String userKey = "login-user:" + ip + "|" + usernameKey(request.username());
        if (rateLimiterService.isLimited(userKey, MAX_FAILURES_PER_USER, WINDOW_SECONDS)) {
            throw new TooManyRequestsException(TOO_MANY_ATTEMPTS);
        }

        LoginResponse response;
        try {
            response = authService.login(request);
        } catch (AuthenticationException ex) {
            rateLimiterService.record(userKey, WINDOW_SECONDS);
            throw ex;
        }
        rateLimiterService.reset(userKey);
        return ResponseEntity.ok(response);
    }

    // Normalized username, or its SHA-256 hex when too long, so limiter keys stay
    // bounded while distinct long usernames still get distinct counters
    private static String usernameKey(String rawUsername) {
        String normalized = rawUsername == null ? "" : rawUsername.trim().toLowerCase(Locale.ROOT);
        if (normalized.length() <= MAX_USERNAME_KEY_LENGTH) {
            return normalized;
        }
        try {
            byte[] digest = MessageDigest.getInstance("SHA-256").digest(normalized.getBytes(StandardCharsets.UTF_8));
            return "sha256:" + HexFormat.of().formatHex(digest);
        } catch (NoSuchAlgorithmException e) {
            // SHA-256 is mandatory on every Java platform
            throw new IllegalStateException("SHA-256 not available", e);
        }
    }
}
