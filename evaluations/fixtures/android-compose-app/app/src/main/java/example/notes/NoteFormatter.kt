package example.notes

object NoteFormatter {
    private const val MAX_TITLE_LENGTH = 40

    /** Trims the note, collapses inner whitespace, and shortens it to a title. */
    fun title(note: String): String {
        val collapsed = note.trim().replace(Regex("\\s+"), " ")
        return if (collapsed.length <= MAX_TITLE_LENGTH) {
            collapsed
        } else {
            collapsed.take(MAX_TITLE_LENGTH - 1) + "…"
        }
    }
}
