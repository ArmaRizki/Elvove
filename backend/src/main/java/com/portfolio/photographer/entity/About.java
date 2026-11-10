package com.portfolio.photographer.entity;

import jakarta.persistence.Column;
import jakarta.persistence.Entity;
import jakarta.persistence.Table;
import lombok.Getter;
import lombok.NoArgsConstructor;
import lombok.Setter;

// Single-row table: admin edits one About record for the whole site.
@Getter
@Setter
@NoArgsConstructor
@Entity
@Table(name = "about_info")
public class About extends BaseEntity {

    @Column(length = 3000)
    private String introduction;

    private Integer yearsOfExperience;

    private String photographyStyle;

    @Column(length = 2000)
    private String achievements;

    @Column(length = 3000)
    private String story;

    private String imageUrl;
}
