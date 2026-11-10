package com.portfolio.photographer.repository;

import com.portfolio.photographer.entity.Photo;
import org.springframework.data.jpa.repository.EntityGraph;
import org.springframework.data.jpa.repository.JpaRepository;

import java.util.List;

/**
 * The list queries fetch the category in the same select. PhotoService reads
 * {@code photo.getCategory()} for every row, and Photo.category is lazy, so
 * without the entity graph each gallery request would issue one extra query per
 * distinct category (an N+1) now that open-in-view is disabled.
 */
public interface PhotoRepository extends JpaRepository<Photo, Long> {

    @EntityGraph(attributePaths = "category")
    List<Photo> findByCategoryIdOrderByDisplayOrderAsc(Long categoryId);

    @EntityGraph(attributePaths = "category")
    List<Photo> findByFeaturedTrueOrderByDisplayOrderAsc();

    @EntityGraph(attributePaths = "category")
    List<Photo> findAllByOrderByDisplayOrderAsc();

    @EntityGraph(attributePaths = "category")
    List<Photo> findTop12ByOrderByCreatedAtDesc();
}
