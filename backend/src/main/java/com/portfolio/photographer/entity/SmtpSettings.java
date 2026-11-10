package com.portfolio.photographer.entity;

import jakarta.persistence.*;

@Entity
@Table(name = "smtp_settings")
public class SmtpSettings {
    @Id private Long id;
    @Column(nullable = false, length = 255) private String host;
    @Column(nullable = false) private Integer port;
    @Column(length = 255) private String username;
    @Column(name = "encrypted_password", columnDefinition = "TEXT") private String encryptedPassword;
    @Column(nullable = false, length = 20) private String encryption;
    @Column(nullable = false) private Boolean authentication;
    @Column(name = "sender_name", nullable = false, length = 150) private String senderName;
    @Column(name = "sender_email", nullable = false, length = 255) private String senderEmail;

    public SmtpSettings() {}
    public Long getId() { return id; }
    public void setId(Long id) { this.id = id; }
    public String getHost() { return host; }
    public void setHost(String host) { this.host = host; }
    public Integer getPort() { return port; }
    public void setPort(Integer port) { this.port = port; }
    public String getUsername() { return username; }
    public void setUsername(String username) { this.username = username; }
    public String getEncryptedPassword() { return encryptedPassword; }
    public void setEncryptedPassword(String value) { this.encryptedPassword = value; }
    public String getEncryption() { return encryption; }
    public void setEncryption(String encryption) { this.encryption = encryption; }
    public Boolean getAuthentication() { return authentication; }
    public void setAuthentication(Boolean authentication) { this.authentication = authentication; }
    public String getSenderName() { return senderName; }
    public void setSenderName(String senderName) { this.senderName = senderName; }
    public String getSenderEmail() { return senderEmail; }
    public void setSenderEmail(String senderEmail) { this.senderEmail = senderEmail; }
}
