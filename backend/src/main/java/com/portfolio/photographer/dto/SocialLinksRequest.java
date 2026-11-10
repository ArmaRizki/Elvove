package com.portfolio.photographer.dto;

import jakarta.validation.constraints.Pattern;

public record SocialLinksRequest(
        @Pattern(regexp = "^$|^https?://.*", message = "Instagram link must be a valid URL")
        String instagram,

        @Pattern(regexp = "^$|^https?://.*", message = "Threads link must be a valid URL")
        String threads,

        @Pattern(regexp = "^$|^https?://.*", message = "TikTok link must be a valid URL")
        String tiktok,

        @Pattern(regexp = "^$|^https?://.*", message = "Facebook link must be a valid URL")
        String facebook,

        @Pattern(regexp = "^$|^https?://.*", message = "YouTube link must be a valid URL")
        String youtube,

        @Pattern(regexp = "^$|^https?://.*", message = "Pinterest link must be a valid URL")
        String pinterest,

        @Pattern(regexp = "^$|^https?://.*", message = "LinkedIn link must be a valid URL")
        String linkedin
) {
}
