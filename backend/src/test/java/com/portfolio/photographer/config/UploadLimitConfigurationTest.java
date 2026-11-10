package com.portfolio.photographer.config;

import org.junit.jupiter.api.Test;
import org.springframework.boot.env.YamlPropertySourceLoader;
import org.springframework.core.env.PropertySource;
import org.springframework.core.io.ClassPathResource;
import org.springframework.util.unit.DataSize;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.List;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

import static org.assertj.core.api.Assertions.assertThat;

/**
 * The upload limit lives in three places: the Spring multipart settings, the
 * check FileStorageService applies, and nginx's client_max_body_size. These
 * tests keep the first two in step and warn when nginx falls behind, which is
 * the combination that made a 5 MB photo fail with an opaque error.
 */
class UploadLimitConfigurationTest {

    private static PropertySource<?> applicationYaml() throws IOException {
        List<PropertySource<?>> sources = new YamlPropertySourceLoader()
                .load("application", new ClassPathResource("application.yml"));
        return sources.get(0);
    }

    private static DataSize size(PropertySource<?> source, String key) {
        // The env placeholder default is what applies when nothing is set:
        // "${UPLOAD_MAX_FILE_SIZE:50MB}" resolves to 50MB.
        String raw = String.valueOf(source.getProperty(key));
        String value = raw.contains(":") ? raw.substring(raw.lastIndexOf(':') + 1, raw.length() - 1) : raw;
        return DataSize.parse(value.trim());
    }

    @Test
    void singleFileLimitIsFiftyMegabytesByDefault() throws IOException {
        PropertySource<?> yaml = applicationYaml();

        assertThat(size(yaml, "spring.servlet.multipart.max-file-size"))
                .isEqualTo(DataSize.ofMegabytes(50));
        assertThat(size(yaml, "spring.servlet.multipart.max-request-size"))
                .isEqualTo(DataSize.ofMegabytes(100));
    }

    @Test
    void serviceLimitMatchesTheMultipartLimit() throws IOException {
        PropertySource<?> yaml = applicationYaml();

        assertThat(size(yaml, "app.upload.max-file-size"))
                .isEqualTo(size(yaml, "spring.servlet.multipart.max-file-size"));
    }

    @Test
    void requestLimitLeavesRoomForASecondFile() throws IOException {
        PropertySource<?> yaml = applicationYaml();

        assertThat(size(yaml, "spring.servlet.multipart.max-request-size"))
                .isGreaterThan(size(yaml, "spring.servlet.multipart.max-file-size"));
    }

    @Test
    void nginxAcceptsAtLeastTheRequestLimit() throws IOException {
        PropertySource<?> yaml = applicationYaml();
        DataSize requestLimit = size(yaml, "spring.servlet.multipart.max-request-size");

        // Absent when the backend is built on its own (Docker build context is
        // ./backend), so the comparison only runs in a full checkout.
        Path nginxConf = Path.of("..", "nginx", "default.conf");
        if (!Files.isRegularFile(nginxConf)) return;

        Matcher matcher = Pattern.compile("client_max_body_size\\s+([0-9]+[kKmMgG]?)")
                .matcher(Files.readString(nginxConf));
        assertThat(matcher.find()).as("client_max_body_size in nginx/default.conf").isTrue();

        // nginx uses bare suffixes (100M), Spring wants a B (100MB).
        String nginxSize = matcher.group(1);
        DataSize nginxLimit = DataSize.parse(
                nginxSize.endsWith("B") || nginxSize.endsWith("b") ? nginxSize : nginxSize + "B");

        assertThat(nginxLimit)
                .as("nginx client_max_body_size must not be smaller than the request limit")
                .isGreaterThanOrEqualTo(requestLimit);
    }
}
