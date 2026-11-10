package com.portfolio.photographer;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;

import java.util.TimeZone;

@SpringBootApplication
public class PhotographerPortfolioApplication {

    /** The studio operates in WIB; booking times are stored and sent in this zone. */
    public static final String APP_TIME_ZONE = "Asia/Jakarta";

    public static void main(String[] args) {
        // Set before Spring starts, so LocalDateTime.now(), Jackson and the
        // JDBC driver all agree on the same zone.
        TimeZone.setDefault(TimeZone.getTimeZone(APP_TIME_ZONE));
        SpringApplication.run(PhotographerPortfolioApplication.class, args);
    }

}
