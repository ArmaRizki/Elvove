package com.portfolio.photographer.service;

import com.portfolio.photographer.dto.LoginRequest;
import com.portfolio.photographer.dto.LoginResponse;
import com.portfolio.photographer.entity.Admin;
import com.portfolio.photographer.repository.AdminRepository;
import com.portfolio.photographer.security.JwtService;
import lombok.RequiredArgsConstructor;
import org.springframework.security.authentication.AuthenticationManager;
import org.springframework.security.authentication.UsernamePasswordAuthenticationToken;
import org.springframework.security.core.userdetails.User;
import org.springframework.stereotype.Service;

@Service
@RequiredArgsConstructor
public class AuthService {

    private final AuthenticationManager authenticationManager;
    private final AdminRepository adminRepository;
    private final JwtService jwtService;

    public LoginResponse login(LoginRequest request) {
        // Throws BadCredentialsException (handled by GlobalExceptionHandler) on failure
        authenticationManager.authenticate(
                new UsernamePasswordAuthenticationToken(request.username(), request.password())
        );

        Admin admin = adminRepository.findByUsername(request.username())
                .orElseThrow();

        String token = jwtService.generateToken(
                new User(admin.getUsername(), admin.getPassword(), java.util.Collections.emptyList())
        );

        return new LoginResponse(token, admin.getUsername(), admin.getEmail());
    }
}
