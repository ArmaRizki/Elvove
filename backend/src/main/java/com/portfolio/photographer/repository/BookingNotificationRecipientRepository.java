package com.portfolio.photographer.repository;

import com.portfolio.photographer.entity.BookingNotificationRecipient;
import org.springframework.data.jpa.repository.JpaRepository;

import java.util.List;

public interface BookingNotificationRecipientRepository extends JpaRepository<BookingNotificationRecipient, Long> {
    List<BookingNotificationRecipient> findAllByOrderByIdAsc();
    boolean existsByEmailIgnoreCase(String email);
}
