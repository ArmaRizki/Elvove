package com.portfolio.photographer.controller;

import com.portfolio.photographer.config.SecurityConfig;
import com.portfolio.photographer.repository.BookingRepository;
import com.portfolio.photographer.security.AdminUserDetailsService;
import com.portfolio.photographer.security.JwtAuthFilter;
import com.portfolio.photographer.security.JwtService;
import com.portfolio.photographer.service.BookingEmailTemplate;
import com.portfolio.photographer.service.BookingService;
import com.portfolio.photographer.service.SmtpMailService;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.WebMvcTest;
import org.springframework.boot.test.mock.mockito.MockBean;
import org.springframework.context.annotation.Import;
import org.springframework.security.test.context.support.WithMockUser;
import org.springframework.test.context.TestPropertySource;
import org.springframework.test.web.servlet.MockMvc;

import java.util.Optional;

import static org.mockito.Mockito.verifyNoInteractions;
import static org.mockito.Mockito.when;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

/**
 * The confirmation-email endpoint lives under /api/admin/**, so the security
 * filter chain must reject anonymous callers before the controller runs.
 */
@WebMvcTest(controllers = BookingWorkflowController.class)
@Import({SecurityConfig.class, JwtAuthFilter.class})
@TestPropertySource(properties = "app.cors.allowed-origins=http://localhost:5173")
class BookingWorkflowSecurityTest {

    @Autowired
    private MockMvc mvc;

    @MockBean
    private BookingRepository repository;
    @MockBean
    private BookingService bookingService;
    @MockBean
    private SmtpMailService smtpMailService;
    @MockBean
    private BookingEmailTemplate emailTemplate;
    @MockBean
    private JwtService jwtService;
    @MockBean
    private AdminUserDetailsService adminUserDetailsService;

    @Test
    void anonymousConfirmationEmailIsRejected() throws Exception {
        mvc.perform(post("/api/admin/bookings/7/confirmation-email"))
                .andExpect(status().isUnauthorized());

        verifyNoInteractions(smtpMailService);
    }

    @Test
    void anonymousAcceptIsRejected() throws Exception {
        mvc.perform(post("/api/admin/bookings/7/accept"))
                .andExpect(status().isUnauthorized());

        verifyNoInteractions(bookingService);
    }

    @Test
    @WithMockUser(roles = "ADMIN")
    void adminReachesTheController() throws Exception {
        when(repository.findById(7L)).thenReturn(Optional.empty());

        // Reaching the controller (and its not-found path) proves authorization
        // passed; an anonymous caller never gets this far.
        mvc.perform(post("/api/admin/bookings/7/confirmation-email"))
                .andExpect(status().isNotFound());
    }
}
