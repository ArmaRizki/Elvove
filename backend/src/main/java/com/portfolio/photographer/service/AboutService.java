package com.portfolio.photographer.service;

import com.portfolio.photographer.dto.AboutRequest;
import com.portfolio.photographer.dto.AboutResponse;
import com.portfolio.photographer.entity.About;
import com.portfolio.photographer.repository.AboutRepository;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

@Service
@RequiredArgsConstructor
public class AboutService {

    private final AboutRepository aboutRepository;

    public AboutResponse get() {
        About about = aboutRepository.findAll().stream().findFirst().orElseGet(About::new);
        return toResponse(about);
    }

    @Transactional
    public AboutResponse update(AboutRequest request) {
        About about = aboutRepository.findAll().stream().findFirst().orElseGet(About::new);
        about.setIntroduction(request.introduction());
        about.setYearsOfExperience(request.yearsOfExperience());
        about.setPhotographyStyle(request.photographyStyle());
        about.setAchievements(request.achievements());
        about.setStory(request.story());
        about.setImageUrl(request.imageUrl());
        return toResponse(aboutRepository.save(about));
    }

    private AboutResponse toResponse(About a) {
        return new AboutResponse(
                a.getId(), a.getIntroduction(), a.getYearsOfExperience(), a.getPhotographyStyle(),
                a.getAchievements(), a.getStory(), a.getImageUrl()
        );
    }
}
