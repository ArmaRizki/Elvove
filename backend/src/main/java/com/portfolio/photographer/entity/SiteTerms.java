package com.portfolio.photographer.entity;

import jakarta.persistence.Column;
import jakarta.persistence.Entity;
import jakarta.persistence.Table;
import lombok.Getter;
import lombok.NoArgsConstructor;
import lombok.Setter;

@Getter
@Setter
@NoArgsConstructor
@Entity
@Table(name = "site_terms")
public class SiteTerms extends BaseEntity {

    @Column(nullable = false, length = 200)
    private String title;

    @Column(length = 1000)
    private String introduction;

    @Column(nullable = false, columnDefinition = "TEXT")
    private String content;

    @Column(nullable = false, length = 20)
    private String version = "1";
}