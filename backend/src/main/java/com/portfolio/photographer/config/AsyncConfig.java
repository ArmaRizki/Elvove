package com.portfolio.photographer.config;

import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.scheduling.annotation.EnableAsync;
import org.springframework.scheduling.concurrent.ThreadPoolTaskExecutor;

import java.util.concurrent.ThreadPoolExecutor;

/**
 * Background executor for administrator booking notifications. The queue is
 * bounded. With AbortPolicy, a full queue throws TaskRejectedException to the
 * caller, which logs the booking ID. Nothing is discarded silently, and memory
 * stays capped. The booking itself is already saved at that point.
 */
@Configuration
@EnableAsync
public class AsyncConfig {

    @Bean(name = "bookingNotificationExecutor")
    public ThreadPoolTaskExecutor bookingNotificationExecutor() {
        ThreadPoolTaskExecutor executor = new ThreadPoolTaskExecutor();
        executor.setCorePoolSize(1);
        executor.setMaxPoolSize(2);
        executor.setQueueCapacity(50);
        executor.setThreadNamePrefix("booking-notify-");
        executor.setRejectedExecutionHandler(new ThreadPoolExecutor.AbortPolicy());
        executor.initialize();
        return executor;
    }
}
