package com.portfolio.photographer.service;

import com.portfolio.photographer.exception.BadRequestException;
import com.portfolio.photographer.exception.PayloadTooLargeException;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;
import org.springframework.util.StringUtils;
import org.springframework.util.unit.DataSize;
import org.springframework.web.multipart.MultipartFile;

import java.io.IOException;
import java.nio.file.*;
import java.util.Locale;
import java.util.Set;
import java.util.UUID;
import java.util.concurrent.TimeUnit;

@Slf4j
@Service
public class FileStorageService {

    private static final Set<String> ALLOWED =
            Set.of("jpg", "jpeg", "png", "webp");

    @Value("${app.upload.dir}")
    private String uploadDir;

    // Configured through UPLOAD_MAX_FILE_SIZE; kept in step with the multipart
    // limit so the rejection message names the same number the container uses.
    @Value("${app.upload.max-file-size:50MB}")
    private DataSize maxFileSize;

    private Path root() {
        return Paths.get(uploadDir).toAbsolutePath().normalize();
    }

    public String store(MultipartFile file) {
        if (file == null || file.isEmpty()) {
            throw new BadRequestException("Tidak ada berkas foto yang diunggah.");
        }

        if (file.getSize() > maxFileSize.toBytes()) {
            throw new PayloadTooLargeException(
                    "Ukuran foto terlalu besar. Maksimal " + maxFileSize.toMegabytes() + " MB.");
        }

        String original = StringUtils.cleanPath(
                file.getOriginalFilename() == null ? "" : file.getOriginalFilename()
        );

        int dot = original.lastIndexOf('.');
        if (dot < 0 || dot == original.length() - 1) {
            throw new BadRequestException("Ekstensi berkas foto tidak valid.");
        }

        String ext = original.substring(dot + 1).toLowerCase(Locale.ROOT);
        if (!ALLOWED.contains(ext)) {
            throw new BadRequestException("Format foto harus JPG, PNG, atau WebP.");
        }

        // The extension alone proves nothing: check the real file signature so a
        // renamed file (for example an HTML or script payload saved as .jpg) is
        // rejected before it is stored and served from /uploads.
        if (!hasValidImageSignature(file, ext)) {
            throw new BadRequestException("Berkas bukan gambar JPG, PNG, atau WebP yang valid.");
        }

        String id = UUID.randomUUID().toString();
        Path originals = root().resolve("originals");
        Path thumbnails = root().resolve("thumbnails");
        Path previews = root().resolve("previews");
        Path source = originals.resolve(id + "." + ext);

        try {
            Files.createDirectories(originals);
            Files.createDirectories(thumbnails);
            Files.createDirectories(previews);

            Files.copy(file.getInputStream(), source);

            convert(source, thumbnails.resolve(id + ".webp"), 800, 55);
            convert(source, previews.resolve(id + ".webp"), 1200, 65);

            return "/uploads/thumbnails/" + id + ".webp";
        } catch (Exception e) {
            deleteById(id);
            log.error("Image processing failed", e);
            throw new BadRequestException("Foto gagal diproses. Pastikan berkas adalah gambar yang valid.");
        }
    }

    public String getThumbnailUrl(String imageUrl) {
        if (imageUrl == null || imageUrl.isBlank()) return null;

        if (imageUrl.startsWith("/uploads/thumbnails/")) {
            return imageUrl;
        }

        // Existing photos: generate WebP variants lazily.
        String filename = safeFilename(imageUrl);
        if (filename == null) return null;

        String id = filename.substring(0, filename.lastIndexOf('.'));
        Path legacy = root().resolve(filename).normalize();
        Path thumbnail = root().resolve("thumbnails").resolve(id + ".webp");

        try {
            if (Files.exists(thumbnail)) {
                return "/uploads/thumbnails/" + id + ".webp";
            }

            if (!legacy.startsWith(root()) || !Files.isRegularFile(legacy)) {
                return null;
            }

            Files.createDirectories(thumbnail.getParent());
            convert(legacy, thumbnail, 800, 55);

            Path preview = root().resolve("previews").resolve(id + ".webp");
            Files.createDirectories(preview.getParent());
            convert(legacy, preview, 1200, 65);

            return "/uploads/thumbnails/" + id + ".webp";
        } catch (Exception e) {
            log.error("Failed to generate legacy thumbnail: {}", imageUrl, e);
            return imageUrl;
        }
    }

    public String getPreviewUrl(String imageUrl) {
        String thumbnail = getThumbnailUrl(imageUrl);
        if (thumbnail == null ||
                !thumbnail.startsWith("/uploads/thumbnails/")) {
            return null;
        }

        String filename = thumbnail.substring(
                "/uploads/thumbnails/".length()
        );

        return "/uploads/previews/" + filename;
    }

    public void delete(String imageUrl) {
        String filename = safeFilename(imageUrl);
        if (filename == null) return;

        String id = filename.substring(0, filename.lastIndexOf('.'));

        deleteById(id);

        // Remove legacy file if this was an old upload.
        try {
            Files.deleteIfExists(root().resolve(filename).normalize());
        } catch (IOException e) {
            log.warn("Could not delete legacy image {}", filename, e);
        }
    }

    /**
     * Reads the first bytes of the upload and matches them against the JPEG, PNG
     * and WebP signatures. cwebp would reject most fakes later, but validating up
     * front keeps the rejected file out of the storage directory entirely.
     */
    private boolean hasValidImageSignature(MultipartFile file, String ext) {
        byte[] header = new byte[12];
        int read;
        try (var in = file.getInputStream()) {
            read = in.readNBytes(header, 0, header.length);
        } catch (IOException e) {
            return false;
        }

        boolean jpeg = read >= 3
                && (header[0] & 0xFF) == 0xFF
                && (header[1] & 0xFF) == 0xD8
                && (header[2] & 0xFF) == 0xFF;

        boolean png = read >= 8
                && (header[0] & 0xFF) == 0x89
                && header[1] == 'P' && header[2] == 'N' && header[3] == 'G'
                && header[4] == 0x0D && header[5] == 0x0A
                && header[6] == 0x1A && header[7] == 0x0A;

        boolean webp = read >= 12
                && header[0] == 'R' && header[1] == 'I' && header[2] == 'F' && header[3] == 'F'
                && header[8] == 'W' && header[9] == 'E' && header[10] == 'B' && header[11] == 'P';

        return switch (ext) {
            case "jpg", "jpeg" -> jpeg;
            case "png" -> png;
            case "webp" -> webp;
            default -> false;
        };
    }

    private String safeFilename(String url) {
        if (url == null || !url.startsWith("/uploads/")) return null;

        String filename = url.substring(url.lastIndexOf('/') + 1);

        if (!filename.matches("[a-zA-Z0-9._-]+") ||
                !filename.contains(".")) {
            return null;
        }

        return filename;
    }

    private void convert(Path source, Path destination,
                         int width, int quality) throws Exception {

        Process process = new ProcessBuilder(
                "cwebp", "-quiet",
                "-resize", String.valueOf(width), "0",
                "-q", String.valueOf(quality),
                source.toString(),
                "-o", destination.toString()
        ).redirectErrorStream(true).start();

        // Larger photos (up to the configured limit) can take longer to resize.
        boolean finished = process.waitFor(120, TimeUnit.SECONDS);

        if (!finished) {
            process.destroyForcibly();
            throw new IOException("WebP conversion timed out");
        }

        if (process.exitValue() != 0 ||
                !Files.exists(destination) ||
                Files.size(destination) == 0) {
            throw new IOException("WebP conversion failed");
        }
    }

    private void deleteById(String id) {
        try {
            Path originals = root().resolve("originals");
            if (Files.isDirectory(originals)) {
                try (var files = Files.list(originals)) {
                    files.filter(p -> p.getFileName().toString().startsWith(id + "."))
                            .forEach(this::deleteQuietly);
                }
            }

            deleteQuietly(root().resolve("thumbnails").resolve(id + ".webp"));
            deleteQuietly(root().resolve("previews").resolve(id + ".webp"));
        } catch (IOException e) {
            log.warn("Failed to clean image files for {}", id, e);
        }
    }

    private void deleteQuietly(Path path) {
        try {
            Files.deleteIfExists(path);
        } catch (IOException e) {
            log.warn("Could not delete {}", path, e);
        }
    }
}
