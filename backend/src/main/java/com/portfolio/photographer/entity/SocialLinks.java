package com.portfolio.photographer.entity;

import jakarta.persistence.Entity;
import jakarta.persistence.Table;
import lombok.Getter;
import lombok.NoArgsConstructor;
import lombok.Setter;

@Getter
@Setter
@NoArgsConstructor
@Entity
@Table(name = "social_links")
public class SocialLinks extends BaseEntity {

    private String instagram;
    private String threads;
    private String tiktok;
    private String facebook;
    private String youtube;
    private String pinterest;
    private String linkedin;
}
