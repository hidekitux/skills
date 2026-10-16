import Foundation

/// Returns the first line of a note, trimmed of surrounding spaces.
func noteTitle(_ text: String) -> String {
    let firstLine = text.split(separator: "\n", omittingEmptySubsequences: false).first ?? ""
    return firstLine.trimmingCharacters(in: .whitespaces)
}
