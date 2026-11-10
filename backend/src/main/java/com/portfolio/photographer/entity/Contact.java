package com.portfolio.photographer.entity;

import jakarta.persistence.Column;
import jakarta.persistence.Entity;
import jakarta.persistence.Table;
import lombok.Getter;
import lombok.NoArgsConstructor;
import lombok.Setter;

// Single-row table: admin edits one Contact record for the whole site.
@Getter
@Setter
@NoArgsConstructor
@Entity
@Table(name = "contact_info")
public class Contact extends BaseEntity {

    private String phone;

    private String email;

    @Column(length = 500)
    private String address;

    @Column(length = 1000)
    private String googleMapLink;

    private String whatsappNumber;
}
