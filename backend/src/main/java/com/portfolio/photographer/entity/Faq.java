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
@Table(name = "faqs")
public class Faq extends BaseEntity {

    @Column(nullable = false)
    private String question;

    @Column(nullable = false, length = 2000)
    private String answer;

    @Column(nullable = false)
    private Integer displayOrder = 0;
}
