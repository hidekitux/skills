final _separators = RegExp(r'[^a-z0-9]+');

/// Lowercases [text] and joins its ASCII letter and digit runs with hyphens.
String slug(String text) => text
    .toLowerCase()
    .split(_separators)
    .where((part) => part.isNotEmpty)
    .join('-');
