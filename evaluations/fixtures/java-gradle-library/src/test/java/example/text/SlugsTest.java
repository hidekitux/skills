package example.text;

import static org.junit.jupiter.api.Assertions.assertEquals;

import org.junit.jupiter.api.Test;

class SlugsTest {
    @Test
    void joinsWords() {
        assertEquals("hello-world", Slugs.slug("Hello World"));
    }

    @Test
    void dropsPunctuation() {
        assertEquals("go-java-rust", Slugs.slug("Go, Java & Rust!"));
    }
}
