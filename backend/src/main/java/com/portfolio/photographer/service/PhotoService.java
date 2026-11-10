package com.portfolio.photographer.service;

import com.portfolio.photographer.dto.PhotoRequest;
import com.portfolio.photographer.dto.PhotoResponse;
import com.portfolio.photographer.entity.Category;
import com.portfolio.photographer.entity.Photo;
import com.portfolio.photographer.exception.ResourceNotFoundException;
import com.portfolio.photographer.repository.CategoryRepository;
import com.portfolio.photographer.repository.PhotoRepository;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.util.List;

@Service
@RequiredArgsConstructor
public class PhotoService {

    private final PhotoRepository photoRepository;
    private final CategoryRepository categoryRepository;
    private final FileStorageService fileStorageService;

    @Transactional(readOnly = true)
    public List<PhotoResponse> getAll() {
        return photoRepository.findAllByOrderByDisplayOrderAsc()
                .stream().map(this::toResponse).toList();
    }

    @Transactional(readOnly = true)
    public List<PhotoResponse> getByCategory(Long categoryId) {
        return photoRepository.findByCategoryIdOrderByDisplayOrderAsc(categoryId)
                .stream().map(this::toResponse).toList();
    }

    @Transactional(readOnly = true)
    public List<PhotoResponse> getFeatured() {
        return photoRepository.findByFeaturedTrueOrderByDisplayOrderAsc()
                .stream().map(this::toResponse).toList();
    }

    @Transactional(readOnly = true)
    public List<PhotoResponse> getLatest() {
        return photoRepository.findTop12ByOrderByCreatedAtDesc()
                .stream().map(this::toResponse).toList();
    }

    @Transactional
    public PhotoResponse create(PhotoRequest request) {
        Category category = categoryRepository.findById(request.categoryId())
                .orElseThrow(() -> new ResourceNotFoundException("Category", request.categoryId()));

        Photo photo = new Photo();
        photo.setTitle(request.title());
        photo.setDescription(request.description());
        photo.setImageUrl(request.imageUrl());
        photo.setCategory(category);
        photo.setFeatured(request.featured());
        photo.setDisplayOrder(request.displayOrder() == null ? 0 : request.displayOrder());

        return toResponse(photoRepository.save(photo));
    }

    @Transactional
    public PhotoResponse update(Long id, PhotoRequest request) {
        Photo photo = photoRepository.findById(id)
                .orElseThrow(() -> new ResourceNotFoundException("Photo", id));

        Category category = categoryRepository.findById(request.categoryId())
                .orElseThrow(() -> new ResourceNotFoundException("Category", request.categoryId()));

        photo.setTitle(request.title());
        photo.setDescription(request.description());
        photo.setImageUrl(request.imageUrl());
        photo.setCategory(category);
        photo.setFeatured(request.featured());
        photo.setDisplayOrder(request.displayOrder() == null
                ? photo.getDisplayOrder() : request.displayOrder());

        return toResponse(photoRepository.save(photo));
    }

    @Transactional
    public void delete(Long id) {
        Photo photo = photoRepository.findById(id)
                .orElseThrow(() -> new ResourceNotFoundException("Photo", id));

        fileStorageService.delete(photo.getImageUrl());
        photoRepository.delete(photo);
    }

    private PhotoResponse toResponse(Photo p) {
        String imageUrl = p.getImageUrl();
        String thumbnailUrl = fileStorageService.getThumbnailUrl(imageUrl);
        String previewUrl = fileStorageService.getPreviewUrl(imageUrl);

        return new PhotoResponse(
                p.getId(), p.getTitle(), p.getDescription(),
                imageUrl, thumbnailUrl, previewUrl,
                p.getCategory() != null ? p.getCategory().getId() : null,
                p.getCategory() != null ? p.getCategory().getName() : null,
                p.isFeatured(), p.getDisplayOrder()
        );
    }
}
