package com.portfolio.photographer.repository;

import com.portfolio.photographer.entity.Contact;
import org.springframework.data.jpa.repository.JpaRepository;

public interface ContactRepository extends JpaRepository<Contact, Long> {
}
