package com.portfolio.photographer.service;

import com.portfolio.photographer.dto.SocialLinksRequest;
import com.portfolio.photographer.dto.SocialLinksResponse;
import com.portfolio.photographer.entity.SocialLinks;
import com.portfolio.photographer.repository.SocialLinksRepository;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

@Service
@RequiredArgsConstructor
public class SocialLinksService {

    private final SocialLinksRepository socialLinksRepository;

    public SocialLinksResponse get() {
        SocialLinks links = socialLinksRepository.findAll()
                .stream()
                .findFirst()
                .orElseGet(SocialLinks::new);

        return toResponse(links);
    }

    @Transactional
    public SocialLinksResponse update(SocialLinksRequest request) {
        SocialLinks links = socialLinksRepository.findAll()
                .stream()
                .findFirst()
                .orElseGet(SocialLinks::new);

        links.setInstagram(request.instagram());
        links.setThreads(request.threads());
        links.setTiktok(request.tiktok());
        links.setFacebook(request.facebook());
        links.setYoutube(request.youtube());
        links.setPinterest(request.pinterest());
        links.setLinkedin(request.linkedin());

        return toResponse(socialLinksRepository.save(links));
    }

    private SocialLinksResponse toResponse(SocialLinks s) {
        return new SocialLinksResponse(
                s.getId(),
                s.getInstagram(),
                s.getThreads(),
                s.getTiktok(),
                s.getFacebook(),
                s.getYoutube(),
                s.getPinterest(),
                s.getLinkedin()
        );
    }
}
