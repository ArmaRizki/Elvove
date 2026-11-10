package com.portfolio.photographer.repository;

import com.portfolio.photographer.entity.Admin;
import org.springframework.data.jpa.repository.JpaRepository;

import java.util.Optional;

public interface AdminRepository extends JpaRepository<Admin, Long> {
    Optional<Admin> findByUsername(String username);
    boolean existsByUsername(String username);
    boolean existsByEmail(String email);
}
