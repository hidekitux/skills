package example.text

/** Lowercases [text] and joins its letter and digit runs with hyphens. */
fun slug(text: String): String =
    text.lowercase()
        .split { !it.isLetterOrDigit() }
        .filter { it.isNotEmpty() }
        .joinToString("-")
