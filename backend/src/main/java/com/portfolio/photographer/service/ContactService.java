package com.portfolio.photographer.service;

import com.portfolio.photographer.dto.ContactRequest;
import com.portfolio.photographer.dto.ContactResponse;
import com.portfolio.photographer.entity.Contact;
import com.portfolio.photographer.repository.ContactRepository;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

@Service
@RequiredArgsConstructor
public class ContactService {

    private final ContactRepository contactRepository;

    public ContactResponse get() {
        Contact contact = contactRepository.findAll().stream().findFirst().orElseGet(Contact::new);
        return toResponse(contact);
    }

    @Transactional
    public ContactResponse update(ContactRequest request) {
        Contact contact = contactRepository.findAll().stream().findFirst().orElseGet(Contact::new);
        contact.setPhone(request.phone());
        contact.setEmail(request.email());
        contact.setAddress(request.address());
        contact.setGoogleMapLink(request.googleMapLink());
        contact.setWhatsappNumber(request.whatsappNumber());
        return toResponse(contactRepository.save(contact));
    }

    private ContactResponse toResponse(Contact c) {
        return new ContactResponse(c.getId(), c.getPhone(), c.getEmail(), c.getAddress(), c.getGoogleMapLink(), c.getWhatsappNumber());
    }
}
