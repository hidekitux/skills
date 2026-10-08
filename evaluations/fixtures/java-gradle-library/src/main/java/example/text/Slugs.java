package example.text;

import java.util.Arrays;
import java.util.Locale;
import java.util.stream.Collectors;

/** Builds URL slugs from free text. */
public final class Slugs {
    private Slugs() {
    }

    /** Lowercases the text and joins its letter and digit runs with hyphens. */
    public static String slug(String text) {
        return Arrays.stream(text.toLowerCase(Locale.ROOT).split("[^\\p{L}\\p{N}]+"))
                .filter(part -> !part.isEmpty())
                .collect(Collectors.joining("-"));
    }
}
