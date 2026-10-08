package example.text

/** Lowercases this string and joins its letter and digit runs with hyphens. */
public fun String.slug(): String =
    lowercase()
        .split(Regex("[^\\p{L}\\p{N}]+"))
        .filter { it.isNotEmpty() }
        .joinToString("-")
