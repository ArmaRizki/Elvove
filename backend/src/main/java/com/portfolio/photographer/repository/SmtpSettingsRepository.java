package com.portfolio.photographer.repository;

import com.portfolio.photographer.entity.SmtpSettings;
import org.springframework.data.jpa.repository.JpaRepository;

public interface SmtpSettingsRepository extends JpaRepository<SmtpSettings, Long> {}
