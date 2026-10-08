package example.notes

import org.junit.Assert.assertEquals
import org.junit.Test

class NoteFormatterTest {
    @Test
    fun collapsesWhitespace() {
        assertEquals("Buy milk", NoteFormatter.title("  Buy \n milk "))
    }

    @Test
    fun shortensLongNotes() {
        val title = NoteFormatter.title("a".repeat(50))
        assertEquals(40, title.length)
        assertEquals('…', title.last())
    }
}
