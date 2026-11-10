package com.portfolio.photographer.service;

import com.portfolio.photographer.dto.CategoryRequest;
import com.portfolio.photographer.dto.CategoryResponse;
import com.portfolio.photographer.entity.Category;
import com.portfolio.photographer.exception.BadRequestException;
import com.portfolio.photographer.exception.ResourceNotFoundException;
import com.portfolio.photographer.repository.CategoryRepository;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.util.List;
import java.util.Locale;

@Service
@RequiredArgsConstructor
public class CategoryService {

    private final CategoryRepository categoryRepository;

    public List<CategoryResponse> getAll() {
        return categoryRepository.findAll().stream().map(this::toResponse).toList();
    }

    @Transactional
    public CategoryResponse create(CategoryRequest request) {
        if (categoryRepository.existsByName(request.name())) {
            throw new BadRequestException("Category '" + request.name() + "' already exists");
        }
        Category category = new Category();
        category.setName(request.name());
        category.setSlug(toSlug(request.name()));
        return toResponse(categoryRepository.save(category));
    }

    @Transactional
    public CategoryResponse update(Long id, CategoryRequest request) {
        Category category = categoryRepository.findById(id)
                .orElseThrow(() -> new ResourceNotFoundException("Category", id));
        category.setName(request.name());
        category.setSlug(toSlug(request.name()));
        return toResponse(categoryRepository.save(category));
    }

    @Transactional
    public void delete(Long id) {
        if (!categoryRepository.existsById(id)) {
            throw new ResourceNotFoundException("Category", id);
        }
        categoryRepository.deleteById(id);
    }

    private String toSlug(String name) {
        return name.toLowerCase(Locale.ROOT).trim().replaceAll("[^a-z0-9]+", "-").replaceAll("(^-|-$)", "");
    }

    private CategoryResponse toResponse(Category c) {
        return new CategoryResponse(c.getId(), c.getName(), c.getSlug());
    }
}
