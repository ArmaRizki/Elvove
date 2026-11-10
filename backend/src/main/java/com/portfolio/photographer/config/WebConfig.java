package com.portfolio.photographer.config;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.Configuration;
import org.springframework.web.servlet.config.annotation.ResourceHandlerRegistry;
import org.springframework.web.servlet.config.annotation.WebMvcConfigurer;

import java.io.File;

@Configuration
public class WebConfig implements WebMvcConfigurer {

    @Value("${app.upload.dir}")
    private String uploadDir;

    @Override
    public void addResourceHandlers(ResourceHandlerRegistry registry) {
        String root = new File(uploadDir).getAbsolutePath() + File.separator;

        registry.addResourceHandler("/uploads/thumbnails/**")
                .addResourceLocations("file:" + root + "thumbnails/");

        registry.addResourceHandler("/uploads/previews/**")
                .addResourceLocations("file:" + root + "previews/");

        registry.addResourceHandler("/uploads/**")
                .addResourceLocations("file:" + root);
    }
}
