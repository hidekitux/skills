package example.text

import kotlin.test.Test
import kotlin.test.assertEquals

class SlugTest {
    @Test
    fun joinsWords() {
        assertEquals("hello-world", slug("Hello World"))
    }

    @Test
    fun dropsPunctuation() {
        assertEquals("go-kotlin-rust", slug("Go, Kotlin & Rust!"))
    }
}
