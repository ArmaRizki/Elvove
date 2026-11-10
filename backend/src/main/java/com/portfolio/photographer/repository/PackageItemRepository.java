package com.portfolio.photographer.repository;

import com.portfolio.photographer.entity.PackageItem;
import org.springframework.data.jpa.repository.JpaRepository;

import java.util.List;

public interface PackageItemRepository extends JpaRepository<PackageItem, Long> {
    List<PackageItem> findAllByOrderByDisplayOrderAsc();
}
