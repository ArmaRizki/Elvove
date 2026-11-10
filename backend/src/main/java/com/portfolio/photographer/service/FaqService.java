package com.portfolio.photographer.service;

import com.portfolio.photographer.dto.FaqRequest;
import com.portfolio.photographer.dto.FaqResponse;
import com.portfolio.photographer.entity.Faq;
import com.portfolio.photographer.exception.ResourceNotFoundException;
import com.portfolio.photographer.repository.FaqRepository;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.util.List;

@Service
@RequiredArgsConstructor
public class FaqService {

    private final FaqRepository faqRepository;

    public List<FaqResponse> getAll() {
        return faqRepository.findAllByOrderByDisplayOrderAsc().stream().map(this::toResponse).toList();
    }

    @Transactional
    public FaqResponse create(FaqRequest request) {
        Faq faq = new Faq();
        apply(faq, request);
        return toResponse(faqRepository.save(faq));
    }

    @Transactional
    public FaqResponse update(Long id, FaqRequest request) {
        Faq faq = faqRepository.findById(id)
                .orElseThrow(() -> new ResourceNotFoundException("Faq", id));
        apply(faq, request);
        return toResponse(faqRepository.save(faq));
    }

    @Transactional
    public void delete(Long id) {
        if (!faqRepository.existsById(id)) {
            throw new ResourceNotFoundException("Faq", id);
        }
        faqRepository.deleteById(id);
    }

    private void apply(Faq faq, FaqRequest request) {
        faq.setQuestion(request.question());
        faq.setAnswer(request.answer());
        faq.setDisplayOrder(request.displayOrder() == null ? 0 : request.displayOrder());
    }

    private FaqResponse toResponse(Faq f) {
        return new FaqResponse(f.getId(), f.getQuestion(), f.getAnswer(), f.getDisplayOrder());
    }
}
