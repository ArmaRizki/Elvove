package com.portfolio.photographer.service;

import com.portfolio.photographer.exception.BadRequestException;
import com.portfolio.photographer.repository.BookingNotificationRecipientRepository;
import com.portfolio.photographer.repository.BookingNotificationSettingsRepository;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.params.ParameterizedTest;
import org.junit.jupiter.params.provider.ValueSource;

import java.util.ArrayList;
import java.util.List;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.inOrder;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

class BookingNotificationSettingsServiceTest {

    private BookingNotificationSettingsService service;

    @BeforeEach
    void setUp() {
        service = new BookingNotificationSettingsService(
                mock(BookingNotificationSettingsRepository.class),
                mock(BookingNotificationRecipientRepository.class));
    }

    @Test
    void trimsAndKeepsSubmissionOrder() {
        assertThat(service.normalize(List.of("  b@contoh.com ", "a@contoh.com")))
                .containsExactly("b@contoh.com", "a@contoh.com");
    }

    @Test
    void nullOrEmptyListMeansNoRecipients() {
        assertThat(service.normalize(null)).isEmpty();
        assertThat(service.normalize(List.of())).isEmpty();
    }

    @Test
    void rejectsDuplicatesCaseInsensitively() {
        assertThatThrownBy(() -> service.normalize(List.of("Admin@Contoh.com", " admin@contoh.COM ")))
                .isInstanceOf(BadRequestException.class)
                .hasMessageContaining("duplikat");
    }

    @ParameterizedTest
    @ValueSource(strings = {"", "   ", "tanpa-at.com", "a@b", "a b@contoh.com", "a@@contoh.com"})
    void rejectsInvalidFormats(String email) {
        assertThatThrownBy(() -> service.normalize(List.of(email)))
                .isInstanceOf(BadRequestException.class);
    }

    @Test
    void rejectsNullEntry() {
        assertThatThrownBy(() -> service.normalize(java.util.Arrays.asList("a@contoh.com", null)))
                .isInstanceOf(BadRequestException.class);
    }

    @Test
    void rejectsMoreThanTwentyRecipients() {
        List<String> many = new ArrayList<>();
        for (int i = 0; i < 21; i++) many.add("admin" + i + "@contoh.com");
        assertThatThrownBy(() -> service.normalize(many))
                .isInstanceOf(BadRequestException.class);
    }

    @Test
    void invalidListIsRejectedBeforeAnyDatabaseWrite() {
        BookingNotificationSettingsRepository settings = mock(BookingNotificationSettingsRepository.class);
        BookingNotificationRecipientRepository recipients = mock(BookingNotificationRecipientRepository.class);
        BookingNotificationSettingsService guarded = new BookingNotificationSettingsService(settings, recipients);

        assertThatThrownBy(() -> guarded.save(true, List.of("a@contoh.com", "A@contoh.com")))
                .isInstanceOf(BadRequestException.class);

        verify(recipients, never()).deleteAllInBatch();
        verify(recipients, never()).saveAll(any());
        verify(settings, never()).save(any());
    }

    // The original 500: Hibernate runs inserts before deletes in a flush, so a
    // plain deleteAll() left the old rows in place and re-inserting a retained
    // address broke the unique index on lower(email). The delete must therefore
    // be a bulk delete that runs before the inserts, and it must be flushed.
    @Test
    void deleteReachesTheDatabaseBeforeRowsAreInserted() {
        BookingNotificationSettingsRepository settings = mock(BookingNotificationSettingsRepository.class);
        BookingNotificationRecipientRepository recipients = mock(BookingNotificationRecipientRepository.class);
        BookingNotificationSettingsService service = new BookingNotificationSettingsService(settings, recipients);

        service.save(true, List.of("tetap@contoh.com", "baru@contoh.com"));

        org.mockito.InOrder order = inOrder(recipients);
        order.verify(recipients).deleteAllInBatch();
        order.verify(recipients).flush();
        order.verify(recipients).saveAll(any());
        order.verify(recipients).flush();
    }

    @Test
    void savingAListThatKeepsAnExistingAddressSucceeds() {
        BookingNotificationSettingsRepository settings = mock(BookingNotificationSettingsRepository.class);
        BookingNotificationRecipientRepository recipients = mock(BookingNotificationRecipientRepository.class);
        BookingNotificationSettingsService service = new BookingNotificationSettingsService(settings, recipients);

        // "tetap@contoh.com" already exists in the database, so this save hits
        // the unique index unless the delete is flushed first.
        BookingNotificationSettingsService.Snapshot saved =
                service.save(true, List.of("tetap@contoh.com", "tambahan@contoh.com"));

        assertThat(saved.enabled()).isTrue();
        assertThat(saved.recipients()).containsExactly("tetap@contoh.com", "tambahan@contoh.com");
        verify(recipients).deleteAllInBatch();
        verify(recipients).saveAll(any());
        verify(settings).save(any());
    }

    @Test
    void removingAnAddressSavesTheRemainingOnes() {
        BookingNotificationSettingsRepository settings = mock(BookingNotificationSettingsRepository.class);
        BookingNotificationRecipientRepository recipients = mock(BookingNotificationRecipientRepository.class);
        BookingNotificationSettingsService service = new BookingNotificationSettingsService(settings, recipients);

        BookingNotificationSettingsService.Snapshot saved =
                service.save(false, List.of("sisa@contoh.com"));

        assertThat(saved.enabled()).isFalse();
        assertThat(saved.recipients()).containsExactly("sisa@contoh.com");
    }

    @Test
    void emptyListClearsAllRecipients() {
        BookingNotificationSettingsRepository settings = mock(BookingNotificationSettingsRepository.class);
        BookingNotificationRecipientRepository recipients = mock(BookingNotificationRecipientRepository.class);
        BookingNotificationSettingsService service = new BookingNotificationSettingsService(settings, recipients);

        BookingNotificationSettingsService.Snapshot saved = service.save(false, List.of());

        assertThat(saved.recipients()).isEmpty();
        verify(recipients).deleteAllInBatch();
    }

    @Test
    void settingsRowIsCreatedWhenItDoesNotExistYet() {
        BookingNotificationSettingsRepository settings = mock(BookingNotificationSettingsRepository.class);
        BookingNotificationRecipientRepository recipients = mock(BookingNotificationRecipientRepository.class);
        when(settings.findById(1L)).thenReturn(java.util.Optional.empty());
        BookingNotificationSettingsService service = new BookingNotificationSettingsService(settings, recipients);

        BookingNotificationSettingsService.Snapshot saved = service.save(true, List.of("a@contoh.com"));

        assertThat(saved.enabled()).isTrue();
        org.mockito.ArgumentCaptor<com.portfolio.photographer.entity.BookingNotificationSettings> captor =
                org.mockito.ArgumentCaptor.forClass(com.portfolio.photographer.entity.BookingNotificationSettings.class);
        verify(settings).save(captor.capture());
        assertThat(captor.getValue().getId()).isEqualTo(1L);
        assertThat(captor.getValue().getEnabled()).isTrue();
        assertThat(captor.getValue().getUpdatedAt()).isNotNull();
    }

    // A constraint violation that slips past validation (race, legacy row) must
    // become a 400 with a readable message, never a generic 500.
    @Test
    void constraintViolationIsReportedAsInvalidInput() {
        BookingNotificationSettingsRepository settings = mock(BookingNotificationSettingsRepository.class);
        BookingNotificationRecipientRepository recipients = mock(BookingNotificationRecipientRepository.class);
        when(recipients.saveAll(any())).thenThrow(
                new org.springframework.dao.DataIntegrityViolationException("duplicate key"));
        BookingNotificationSettingsService service = new BookingNotificationSettingsService(settings, recipients);

        assertThatThrownBy(() -> service.save(true, List.of("a@contoh.com")))
                .isInstanceOf(BadRequestException.class)
                .hasMessageContaining("duplikat");
        verify(settings, never()).save(any());
    }

    @Test
    void currentReturnsEmptyDisabledStateWhenNothingSaved() {
        BookingNotificationSettingsRepository settings = mock(BookingNotificationSettingsRepository.class);
        BookingNotificationRecipientRepository recipients = mock(BookingNotificationRecipientRepository.class);
        when(settings.findById(1L)).thenReturn(java.util.Optional.empty());
        when(recipients.findAllByOrderByIdAsc()).thenReturn(List.of());

        BookingNotificationSettingsService fresh = new BookingNotificationSettingsService(settings, recipients);
        BookingNotificationSettingsService.Snapshot snapshot = fresh.current();

        assertThat(snapshot.enabled()).isFalse();
        assertThat(snapshot.recipients()).isEmpty();
    }
}
