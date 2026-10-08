/// Lowercases `text` and joins its letter and digit runs with hyphens.
public func slug(_ text: String) -> String {
    text.lowercased()
        .split(whereSeparator: { !$0.isLetter && !$0.isNumber })
        .joined(separator: "-")
}
