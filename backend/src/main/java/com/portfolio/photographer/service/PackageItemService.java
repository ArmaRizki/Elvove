package com.portfolio.photographer.service;

import com.portfolio.photographer.dto.PackageItemRequest;
import com.portfolio.photographer.dto.PackageItemResponse;
import com.portfolio.photographer.entity.PackageItem;
import com.portfolio.photographer.exception.ResourceNotFoundException;
import com.portfolio.photographer.repository.PackageItemRepository;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.util.ArrayList;
import java.util.List;

@Service
@RequiredArgsConstructor
public class PackageItemService {

    private final PackageItemRepository packageItemRepository;

    public List<PackageItemResponse> getAll() {
        return packageItemRepository.findAllByOrderByDisplayOrderAsc().stream().map(this::toResponse).toList();
    }

    @Transactional
    public PackageItemResponse create(PackageItemRequest request) {
        PackageItem item = new PackageItem();
        apply(item, request);
        return toResponse(packageItemRepository.save(item));
    }

    @Transactional
    public PackageItemResponse update(Long id, PackageItemRequest request) {
        PackageItem item = packageItemRepository.findById(id)
                .orElseThrow(() -> new ResourceNotFoundException("Package", id));
        apply(item, request);
        return toResponse(packageItemRepository.save(item));
    }

    @Transactional
    public void delete(Long id) {
        if (!packageItemRepository.existsById(id)) {
            throw new ResourceNotFoundException("Package", id);
        }
        packageItemRepository.deleteById(id);
    }

    private void apply(PackageItem item, PackageItemRequest request) {
        item.setTitle(request.title());
        item.setPrice(request.price());
        item.setFeatures(request.features() == null ? new ArrayList<>() : request.features());
        item.setDisplayOrder(request.displayOrder() == null ? 0 : request.displayOrder());
        item.setHighlighted(request.highlighted());
    }

    private PackageItemResponse toResponse(PackageItem p) {
        return new PackageItemResponse(p.getId(), p.getTitle(), p.getPrice(), p.getFeatures(), p.getDisplayOrder(), p.isHighlighted());
    }
}
