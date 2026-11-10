package com.portfolio.photographer.security;

import jakarta.servlet.http.HttpServletRequest;
import org.springframework.stereotype.Component;

/**
 * Resolves the client IP used for rate limiting.
 * <p>
 * Trusts only {@code X-Real-IP}, which nginx sets to {@code $remote_addr} and so
 * overwrites any client-supplied value. {@code X-Forwarded-For} is deliberately
 * ignored: nginx appends to it ({@code $proxy_add_x_forwarded_for}), so its first
 * entry is client-controlled and would let callers rotate their rate-limit key.
 */
@Component
public class ClientIpResolver {

    static final int MAX_LENGTH = 64;

    public String resolve(HttpServletRequest request) {
        String realIp = request.getHeader("X-Real-IP");
        String ip = (realIp != null && !realIp.isBlank()) ? realIp : request.getRemoteAddr();
        if (ip == null) {
            return "unknown";
        }
        ip = ip.trim();
        if (ip.isEmpty()) {
            return "unknown";
        }
        // Cap length so a crafted header can't bloat rate-limiter keys
        return ip.length() > MAX_LENGTH ? ip.substring(0, MAX_LENGTH) : ip;
    }
}
