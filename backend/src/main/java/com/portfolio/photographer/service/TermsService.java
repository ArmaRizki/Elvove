package com.portfolio.photographer.service;

import com.portfolio.photographer.dto.TermsRequest;
import com.portfolio.photographer.dto.TermsResponse;
import com.portfolio.photographer.entity.SiteTerms;
import com.portfolio.photographer.repository.SiteTermsRepository;
import lombok.RequiredArgsConstructor;
import org.springframework.http.HttpStatus;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.web.server.ResponseStatusException;

@Service
@RequiredArgsConstructor
public class TermsService {

    private final SiteTermsRepository repository;

    private static final String DEFAULT_CONTENT = """
# Booking & Reservation

- All bookings must be submitted through the official ELVOVE booking form.
- A booking is confirmed only after ELVOVE has approved the requested schedule and received the required down payment (DP).
- Submitting a booking inquiry does not automatically guarantee availability.

# Payment & Down Payment

- A down payment is required to secure a confirmed photography session.
- The down payment is non-refundable if the client cancels the session unilaterally.
- The remaining balance must be paid by the due date stated in the invoice.
- Additional services or charges outside the selected package will be communicated to the client.

# Rescheduling & Cancellation

- Rescheduling requests must be submitted at least three (3) days before the scheduled session.
- All rescheduling requests are subject to photographer availability.
- Requests made less than three (3) days before the session may be declined or subject to additional arrangements.
- If ELVOVE is unable to fulfill a confirmed booking, an alternative schedule or appropriate resolution will be discussed with the client.

# Photography Session

- Clients are expected to arrive on time and be ready at the agreed session location.
- Late arrival may reduce the available photography duration without extending the original session end time.
- Additional session time is subject to photographer availability and may incur overtime charges.
- ELVOVE will make reasonable creative and technical decisions during the session to achieve the intended photographic results.

# Package & Photo Editing

- Photography sessions may include unlimited captured shots, but the number of edited photographs is determined by the selected package.
- Clients may select photographs for editing according to the quantity included in their package.
- Additional edited photographs or special editing requests may incur extra charges.
- Final editing follows ELVOVE's photographic style and agreed package specifications.

# Photo Delivery

- Final photographs will be delivered digitally through a cloud storage link or another agreed delivery method.
- Delivery timelines will be communicated according to the selected package and project requirements.
- Clients are responsible for downloading and safely backing up their delivered photographs.
- The availability period of the download link will be communicated by ELVOVE.

# Client Responsibilities & Additional Costs

- Makeup artists, outfits, accessories, and personal props are the client's responsibility unless explicitly included in the package.
- Location entrance fees, photography permits, parking fees, and other location-related charges are the client's responsibility unless otherwise agreed.
- Clients are responsible for obtaining any required permissions for their chosen photography location.
- Clients should take reasonable care of their personal belongings during the session.

# Weather & Force Majeure

- If severe weather or circumstances beyond reasonable control make the session unsafe or impractical, ELVOVE and the client will discuss an alternative schedule.
- Weather-related rescheduling agreed upon by both parties will not incur an additional rescheduling fee.
- Alternative dates remain subject to photographer availability.

# Copyright & Portfolio Usage

- Copyright and usage rights are governed by applicable law and any separate written agreement between ELVOVE and the client.
- ELVOVE may use selected photographs for portfolios, websites, social media, and promotional materials.
- Clients who do not wish their photographs to be published must notify ELVOVE before the photography session.
- ELVOVE will respect any publication restrictions expressly agreed upon with the client.

# Agreement

- By submitting a booking inquiry and accepting these Terms & Conditions, the client acknowledges having read and understood the applicable booking policies.
- A confirmed booking remains subject to the agreed package, invoice, and any additional written arrangements.
- Any changes to the terms applicable to a confirmed booking should be communicated and agreed upon by the relevant parties.
""";

    @Transactional
    public TermsResponse get() {
        SiteTerms terms = repository.findAll()
                .stream()
                .findFirst()
                .orElseGet(this::createDefault);

        return toResponse(terms);
    }

    @Transactional
    public TermsResponse update(TermsRequest request) {
        SiteTerms terms = repository.findAll()
                .stream()
                .findFirst()
                .orElseGet(this::createDefault);

        boolean changed =
                !terms.getTitle().equals(request.title()) ||
                !safe(terms.getIntroduction()).equals(safe(request.introduction())) ||
                !terms.getContent().equals(request.content());

        if (changed) {
            int currentVersion = Integer.parseInt(terms.getVersion());
            terms.setVersion(String.valueOf(currentVersion + 1));
        }

        terms.setTitle(request.title());
        terms.setIntroduction(request.introduction());
        terms.setContent(request.content());

        return toResponse(repository.save(terms));
    }

    @Transactional(readOnly = true)
    public String getCurrentVersion() {
        SiteTerms terms = repository.findAll()
                .stream()
                .findFirst()
                .orElseGet(this::createDefault);

        return terms.getVersion();
    }

    public void validateVersion(String submittedVersion) {
        String currentVersion = getCurrentVersion();

        if (!currentVersion.equals(submittedVersion)) {
            throw new ResponseStatusException(
                    HttpStatus.CONFLICT,
                    "Terms & Conditions have changed. Please review the latest version."
            );
        }
    }

    private SiteTerms createDefault() {
        SiteTerms terms = new SiteTerms();
        terms.setTitle("Terms & Conditions");
        terms.setIntroduction(
                "Please read these terms carefully before submitting your photography booking inquiry."
        );
        terms.setContent(DEFAULT_CONTENT);
        terms.setVersion("1");

        return repository.save(terms);
    }

    private String safe(String value) {
        return value == null ? "" : value;
    }

    private TermsResponse toResponse(SiteTerms terms) {
        return new TermsResponse(
                terms.getId(),
                terms.getTitle(),
                terms.getIntroduction(),
                terms.getContent(),
                terms.getVersion(),
                terms.getUpdatedAt()
        );
    }
}