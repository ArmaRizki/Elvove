package com.portfolio.photographer.service;

import com.portfolio.photographer.exception.BadRequestException;
import com.portfolio.photographer.exception.PayloadTooLargeException;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.TempDir;
import org.springframework.mock.web.MockMultipartFile;
import org.springframework.test.util.ReflectionTestUtils;
import org.springframework.util.unit.DataSize;
import org.springframework.web.multipart.MultipartFile;

import java.io.InputStream;
import java.nio.file.Files;
import java.nio.file.Path;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

/**
 * Covers the checks FileStorageService applies itself: an empty upload, the
 * configured size limit and the allowed extensions. The size check runs before
 * the file is read, so these tests never buffer a real 50 MB photo.
 */
class FileStorageServiceTest {

    private static final byte[] JPEG_HEADER =
            new byte[] {(byte) 0xFF, (byte) 0xD8, (byte) 0xFF, (byte) 0xE0, 0, 0, 0, 0, 0, 0, 0, 0};
    private static final byte[] PNG_HEADER =
            new byte[] {(byte) 0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0, 0, 0, 0};
    private static final byte[] WEBP_HEADER =
            new byte[] {'R', 'I', 'F', 'F', 0, 0, 0, 0, 'W', 'E', 'B', 'P'};

    @TempDir
    Path uploadDir;

    private FileStorageService service;

    @BeforeEach
    void setUp() {
        service = new FileStorageService();
        ReflectionTestUtils.setField(service, "uploadDir", uploadDir.toString());
        ReflectionTestUtils.setField(service, "maxFileSize", DataSize.ofMegabytes(50));
    }

    @Test
    void emptyFileIsRejected() {
        MockMultipartFile empty = new MockMultipartFile("file", "photo.jpg", "image/jpeg", new byte[0]);

        assertThatThrownBy(() -> service.store(empty))
                .isInstanceOf(BadRequestException.class)
                .hasMessageContaining("Tidak ada berkas foto");
    }

    @Test
    void nullFileIsRejected() {
        assertThatThrownBy(() -> service.store(null))
                .isInstanceOf(BadRequestException.class)
                .hasMessageContaining("Tidak ada berkas foto");
    }

    @Test
    void fileOverTheConfiguredLimitIsRejected() {
        // 50 MB + 1 byte; the service rejects it before reading any content.
        MultipartFile tooBig = stubFile("big.jpg", DataSize.ofMegabytes(50).toBytes() + 1, JPEG_HEADER);

        assertThatThrownBy(() -> service.store(tooBig))
                .isInstanceOf(PayloadTooLargeException.class)
                .hasMessage("Ukuran foto terlalu besar. Maksimal 50 MB.");
    }

    @Test
    void fileAtTheConfiguredLimitPassesTheSizeCheck() {
        // Exactly at the limit is allowed; only the size check is under test
        // here, so the failure that follows must come from image processing.
        MultipartFile atLimit = stubFile("limit.jpg", DataSize.ofMegabytes(50).toBytes(), JPEG_HEADER);

        assertThatThrownBy(() -> service.store(atLimit))
                .isInstanceOf(BadRequestException.class)
                .hasMessageContaining("Foto gagal diproses");
    }

    @Test
    void contentThatIsNotAnImageIsRejectedEvenWithAnImageExtension() {
        // An HTML payload renamed to .jpg must never reach the storage directory.
        MockMultipartFile fake = new MockMultipartFile(
                "file", "photo.jpg", "image/jpeg",
                "<html><script>alert(1)</script></html>".getBytes(java.nio.charset.StandardCharsets.UTF_8));

        assertThatThrownBy(() -> service.store(fake))
                .isInstanceOf(BadRequestException.class)
                .hasMessageContaining("bukan gambar");

        assertThat(uploadDir.resolve("originals")).doesNotExist();
    }

    @Test
    void pngBytesWithAJpgExtensionAreRejected() {
        // The signature must match the declared extension, so a mislabelled file
        // cannot be served later with the wrong content type.
        MockMultipartFile mismatched = new MockMultipartFile(
                "file", "photo.jpg", "image/jpeg", PNG_HEADER);

        assertThatThrownBy(() -> service.store(mismatched))
                .isInstanceOf(BadRequestException.class)
                .hasMessageContaining("bukan gambar");
    }

    @Test
    void webpSignatureIsRecognised() {
        // Reaching image processing proves the signature was accepted.
        MockMultipartFile webp = new MockMultipartFile(
                "file", "photo.webp", "image/webp", WEBP_HEADER);

        assertThatThrownBy(() -> service.store(webp))
                .isInstanceOf(BadRequestException.class)
                .hasMessageContaining("Foto gagal diproses");
    }

    @Test
    void configuredLimitIsUsedInsteadOfAHardcodedOne() {
        ReflectionTestUtils.setField(service, "maxFileSize", DataSize.ofMegabytes(2));
        MultipartFile threeMb = stubFile("photo.jpg", DataSize.ofMegabytes(3).toBytes(), JPEG_HEADER);

        assertThatThrownBy(() -> service.store(threeMb))
                .isInstanceOf(PayloadTooLargeException.class)
                .hasMessage("Ukuran foto terlalu besar. Maksimal 2 MB.");
    }

    @Test
    void unsupportedExtensionIsRejected() {
        MockMultipartFile gif = new MockMultipartFile(
                "file", "photo.gif", "image/gif", new byte[] {1, 2, 3});

        assertThatThrownBy(() -> service.store(gif))
                .isInstanceOf(BadRequestException.class)
                .hasMessageContaining("JPG, PNG, atau WebP");
    }

    @Test
    void fileWithoutExtensionIsRejected() {
        MockMultipartFile noExtension = new MockMultipartFile(
                "file", "photo", "image/jpeg", new byte[] {1, 2, 3});

        assertThatThrownBy(() -> service.store(noExtension))
                .isInstanceOf(BadRequestException.class)
                .hasMessageContaining("Ekstensi berkas foto tidak valid");
    }

    @Test
    void traversalInTheOriginalNameCannotEscapeTheUploadDirectory() {
        Path outside = uploadDir.getParent().resolve("escaped.jpg");
        MockMultipartFile sneaky = new MockMultipartFile(
                "file", "../../escaped.jpg", "image/jpeg", JPEG_HEADER);

        assertThatThrownBy(() -> service.store(sneaky))
                .isInstanceOf(BadRequestException.class);

        assertThat(Files.exists(outside)).isFalse();
        assertThat(uploadDir.resolve("originals")).isDirectory();
    }

    /**
     * A MultipartFile that only reports a size. Reading is never reached because
     * the size check comes first, so no large buffer is allocated.
     */
    private static MultipartFile stubFile(String name, long size, byte[] header) {
        return new MultipartFile() {
            @Override
            public String getName() {
                return "file";
            }

            @Override
            public String getOriginalFilename() {
                return name;
            }

            @Override
            public String getContentType() {
                return "image/jpeg";
            }

            @Override
            public boolean isEmpty() {
                return false;
            }

            @Override
            public long getSize() {
                return size;
            }

            @Override
            public byte[] getBytes() {
                throw new UnsupportedOperationException();
            }

            @Override
            public InputStream getInputStream() {
                // Signature is checked, then image processing fails on this stub.
                return new java.io.ByteArrayInputStream(header);
            }

            @Override
            public void transferTo(java.io.File dest) {
                throw new UnsupportedOperationException();
            }
        };
    }
}
