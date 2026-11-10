package com.portfolio.photographer.config;

import com.portfolio.photographer.entity.Admin;
import com.portfolio.photographer.repository.AdminRepository;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.params.ParameterizedTest;
import org.junit.jupiter.params.provider.ValueSource;
import org.mockito.ArgumentCaptor;
import org.springframework.security.crypto.bcrypt.BCryptPasswordEncoder;
import org.springframework.security.crypto.password.PasswordEncoder;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.*;

class DataInitializerTest {

    private static final String VALID_PASSWORD = "Str0ng-Unique-Passphrase";

    private AdminRepository repository;
    private final PasswordEncoder encoder = new BCryptPasswordEncoder();

    @BeforeEach
    void setUp() {
        repository = mock(AdminRepository.class);
    }

    private DataInitializer init(String username, String email, String password) {
        return new DataInitializer(repository, encoder, username, email, password);
    }

    @Test
    void emptyTableWithValidConfigCreatesExactlyOneAdmin() {
        when(repository.count()).thenReturn(0L);

        init("  owner  ", "owner@example.com", VALID_PASSWORD).run();

        ArgumentCaptor<Admin> captor = ArgumentCaptor.forClass(Admin.class);
        verify(repository, times(1)).saveAndFlush(captor.capture());
        Admin saved = captor.getValue();
        assertThat(saved.getUsername()).isEqualTo("owner");
        assertThat(saved.getEmail()).isEqualTo("owner@example.com");
        assertThat(saved.getPassword()).isNotEqualTo(VALID_PASSWORD).startsWith("$2");
        assertThat(encoder.matches(VALID_PASSWORD, saved.getPassword())).isTrue();
        verify(repository).count();
        verifyNoMoreInteractions(repository);
    }

    @Test
    void existingAdminIsNeverTouched() {
        when(repository.count()).thenReturn(1L);

        // Even with weak/missing config, nothing happens when an admin exists
        init("", "", "").run();
        init("admin", "admin@example.com", "admin123").run();

        verify(repository, times(2)).count();
        verifyNoMoreInteractions(repository);
    }

    @Test
    void missingPasswordFailsWithoutSaving() {
        when(repository.count()).thenReturn(0L);

        assertThatThrownBy(() -> init("owner", "owner@example.com", "").run())
                .isInstanceOf(IllegalStateException.class)
                .hasMessageContaining("ADMIN_PASSWORD");
        assertThatThrownBy(() -> init("owner", "owner@example.com", null).run())
                .isInstanceOf(IllegalStateException.class);
        verify(repository, never()).saveAndFlush(any());
    }

    @Test
    void missingUsernameFails() {
        when(repository.count()).thenReturn(0L);

        assertThatThrownBy(() -> init("   ", "owner@example.com", VALID_PASSWORD).run())
                .isInstanceOf(IllegalStateException.class)
                .hasMessageContaining("ADMIN_USERNAME");
        verify(repository, never()).saveAndFlush(any());
    }

    @ParameterizedTest
    @ValueSource(strings = {"admin123", "ChangeMe123!", "CHANGEME123!", "PASSWORD123", "elevenChars"})
    void weakOrDefaultPasswordsFail(String weak) {
        when(repository.count()).thenReturn(0L);

        assertThatThrownBy(() -> init("owner", "owner@example.com", weak).run())
                .isInstanceOf(IllegalStateException.class)
                .hasMessageContaining("ADMIN_PASSWORD")
                .satisfies(ex -> assertThat(ex.getMessage()).doesNotContain(weak));
        verify(repository, never()).saveAndFlush(any());
    }

    @ParameterizedTest
    @ValueSource(strings = {"", "not-an-email", "a@b", "two@@example.com", "sp ace@example.com"})
    void invalidEmailFails(String email) {
        when(repository.count()).thenReturn(0L);

        assertThatThrownBy(() -> init("owner", email, VALID_PASSWORD).run())
                .isInstanceOf(IllegalStateException.class)
                .hasMessageContaining("ADMIN_EMAIL");
        verify(repository, never()).saveAndFlush(any());
    }

    @Test
    void exceptionMessageNeverContainsPassword() {
        when(repository.count()).thenReturn(0L);
        String secretButShort = "s3cr3tVal";

        assertThatThrownBy(() -> init("owner", "owner@example.com", secretButShort).run())
                .isInstanceOf(IllegalStateException.class)
                .satisfies(ex -> assertThat(ex.getMessage()).doesNotContain(secretButShort));
        // Invalid email with valid password: message must not leak the password either
        assertThatThrownBy(() -> init("owner", "bad", VALID_PASSWORD).run())
                .satisfies(ex -> assertThat(ex.getMessage()).doesNotContain(VALID_PASSWORD));
    }
}
