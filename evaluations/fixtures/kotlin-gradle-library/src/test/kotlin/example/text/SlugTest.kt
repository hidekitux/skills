package example.text

import kotlin.test.Test
import kotlin.test.assertEquals

class SlugTest {
    @Test
    fun joinsWords() {
        assertEquals("hello-world", "Hello World".slug())
    }

    @Test
    fun dropsPunctuation() {
        assertEquals("go-kotlin-rust", "Go, Kotlin & Rust!".slug())
    }
}
