package com.portfolio.photographer.entity;

import jakarta.persistence.Column;
import jakarta.persistence.Entity;
import jakarta.persistence.Table;
import lombok.Getter;
import lombok.NoArgsConstructor;
import lombok.Setter;

// e.g. Wedding, Pre-Wedding, Engagement, Birthday, Baby Shoot, Maternity,
// Fashion, Product Photography, Travel, Wildlife, Events
@Getter
@Setter
@NoArgsConstructor
@Entity
@Table(name = "categories")
public class Category extends BaseEntity {

    @Column(nullable = false, unique = true)
    private String name;

    @Column(unique = true)
    private String slug;
}
