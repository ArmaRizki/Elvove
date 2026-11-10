package com.portfolio.photographer.entity;

import jakarta.persistence.CollectionTable;
import jakarta.persistence.Column;
import jakarta.persistence.ElementCollection;
import jakarta.persistence.Entity;
import jakarta.persistence.FetchType;
import jakarta.persistence.JoinColumn;
import jakarta.persistence.Table;
import lombok.Getter;
import lombok.NoArgsConstructor;
import lombok.Setter;

import java.math.BigDecimal;
import java.util.ArrayList;
import java.util.List;

// Named PackageItem to avoid clashing with java.lang.Package
@Getter
@Setter
@NoArgsConstructor
@Entity
@Table(name = "packages")
public class PackageItem extends BaseEntity {

    @Column(nullable = false)
    private String title; // e.g. Basic, Premium, Luxury

    @Column(nullable = false, precision = 10, scale = 2)
    private BigDecimal price;

    @ElementCollection(fetch = FetchType.EAGER)
    @CollectionTable(name = "package_features", joinColumns = @JoinColumn(name = "package_id"))
    @Column(name = "feature")
    private List<String> features = new ArrayList<>();

    @Column(nullable = false)
    private Integer displayOrder = 0;

    @Column(nullable = false)
    private boolean highlighted = false; // e.g. mark "Most Popular"
}
