package com.portfolio.photographer.service;

import com.portfolio.photographer.entity.SmtpSettings;
import com.portfolio.photographer.repository.SmtpSettingsRepository;
import org.junit.jupiter.api.Test;

import java.io.BufferedReader;
import java.io.BufferedWriter;
import java.io.IOException;
import java.io.InputStreamReader;
import java.io.OutputStreamWriter;
import java.io.Writer;
import java.net.InetAddress;
import java.net.ServerSocket;
import java.net.Socket;
import java.nio.charset.StandardCharsets;
import java.util.Locale;
import java.util.Optional;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;
import static org.mockito.ArgumentMatchers.anyLong;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.when;

class SmtpMailServiceTest {

    private final SmtpSettingsRepository repository = mock(SmtpSettingsRepository.class);

    @Test
    void blankKeyFails() {
        assertThatThrownBy(() -> new SmtpMailService(repository, ""))
                .isInstanceOf(IllegalStateException.class).hasMessageContaining("SMTP_ENCRYPTION_KEY");
        assertThatThrownBy(() -> new SmtpMailService(repository, null))
                .isInstanceOf(IllegalStateException.class);
    }

    @Test
    void shortKeyFails() {
        assertThatThrownBy(() -> new SmtpMailService(repository, "x".repeat(31)))
                .isInstanceOf(IllegalStateException.class)
                .hasMessage("SMTP_ENCRYPTION_KEY must be at least 32 bytes");
    }

    @Test
    void validKeyEncrypts() {
        SmtpMailService service = new SmtpMailService(repository, "k".repeat(64));
        String encrypted = service.encryptPassword("smtp-pass");
        assertThat(encrypted).isNotBlank().doesNotContain("smtp-pass");
    }

    @Test
    void htmlMailIsDeliveredAsMultipartAlternative() throws Exception {
        try (FakeSmtpServer server = new FakeSmtpServer()) {
            SmtpMailService service = serviceFor(server.port());
            String subject = "Your Booking Has Been Received - Elvove";

            service.sendHtml("customer@example.com", subject,
                    "<html><body><p>Hello, Anindya Putri.</p></body></html>",
                    "Hello, Anindya Putri.");

            String raw = server.receivedMessage();
            assertThat(raw)
                    .contains("Subject: " + subject)
                    .contains("customer@example.com")
                    .contains("elvo.vee@gmail.com")
                    .contains("Elvove")
                    .contains("multipart/alternative")
                    .contains("text/html")
                    .contains("text/plain")
                    .contains("Hello, Anindya Putri.");
        }
    }

    @Test
    void failedDeliveryIsReportedWithoutLeakingCredentials() throws Exception {
        SmtpMailService service = new SmtpMailService(repository, "k".repeat(64));
        when(repository.findById(anyLong())).thenReturn(Optional.empty());

        assertThatThrownBy(() -> service.sendHtml("customer@example.com", "Subject",
                "<p>body</p>", "body"))
                .isInstanceOf(IllegalStateException.class)
                .hasMessageContaining("Gagal mengirim email SMTP")
                .hasMessageNotContaining("k".repeat(64))
                .hasMessageNotContaining("password");
    }

    @Test
    void anUnreachableServerFailsTheSendInsteadOfReportingSuccess() throws Exception {
        int closedPort;
        try (ServerSocket probe = new ServerSocket(0, 1, InetAddress.getLoopbackAddress())) {
            closedPort = probe.getLocalPort();
        }
        SmtpMailService service = serviceFor(closedPort);

        assertThatThrownBy(() -> service.sendHtml("customer@example.com", "Subject",
                "<p>body</p>", "body"))
                .isInstanceOf(IllegalStateException.class)
                .hasMessageContaining("Gagal mengirim email SMTP");
    }

    private SmtpMailService serviceFor(int port) {
        SmtpSettings settings = new SmtpSettings();
        settings.setId(1L);
        settings.setHost("127.0.0.1");
        settings.setPort(port);
        settings.setUsername("");
        settings.setEncryptedPassword(null);
        settings.setEncryption("NONE");
        settings.setAuthentication(false);
        settings.setSenderName("Elvove");
        settings.setSenderEmail("elvo.vee@gmail.com");
        when(repository.findById(1L)).thenReturn(Optional.of(settings));
        return new SmtpMailService(repository, "k".repeat(64));
    }

    /**
     * A minimal SMTP server that accepts one message and keeps its raw
     * content, so the delivered MIME structure can be inspected without a
     * real mail server.
     */
    private static final class FakeSmtpServer implements AutoCloseable {
        private final ServerSocket socket;
        private final Thread thread;
        private volatile String message;

        FakeSmtpServer() throws IOException {
            socket = new ServerSocket(0, 1, InetAddress.getLoopbackAddress());
            thread = new Thread(this::serve);
            thread.setDaemon(true);
            thread.start();
        }

        int port() {
            return socket.getLocalPort();
        }

        String receivedMessage() throws InterruptedException {
            thread.join(15_000);
            assertThat(message).as("the SMTP server received a message").isNotNull();
            return message;
        }

        private void serve() {
            try (Socket client = socket.accept();
                 BufferedReader in = new BufferedReader(
                         new InputStreamReader(client.getInputStream(), StandardCharsets.UTF_8));
                 Writer out = new BufferedWriter(
                         new OutputStreamWriter(client.getOutputStream(), StandardCharsets.UTF_8))) {
                out.write("220 localhost ESMTP test\r\n");
                out.flush();

                boolean receiving = false;
                StringBuilder data = new StringBuilder();
                String line;
                while ((line = in.readLine()) != null) {
                    if (receiving) {
                        if (".".equals(line)) {
                            receiving = false;
                            message = data.toString();
                            out.write("250 OK queued\r\n");
                            out.flush();
                            continue;
                        }
                        data.append(line).append('\n');
                        continue;
                    }
                    String command = line.toUpperCase(Locale.ROOT);
                    if (command.startsWith("EHLO") || command.startsWith("HELO")) {
                        out.write("250 localhost\r\n");
                    } else if (command.startsWith("DATA")) {
                        out.write("354 End data with <CR><LF>.<CR><LF>\r\n");
                        receiving = true;
                        data.setLength(0);
                    } else if (command.startsWith("QUIT")) {
                        out.write("221 Bye\r\n");
                        out.flush();
                        break;
                    } else {
                        out.write("250 OK\r\n");
                    }
                    out.flush();
                }
            } catch (IOException ignored) {
                // The socket is closed when the test finishes.
            }
        }

        @Override
        public void close() throws IOException {
            socket.close();
        }
    }
}
