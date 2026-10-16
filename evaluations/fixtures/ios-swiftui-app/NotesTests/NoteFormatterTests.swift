import Testing
@testable import Notes

@Test(arguments: [
    ("  Buy milk  \nfrom the corner shop", "Buy milk"),
    ("Call the dentist", "Call the dentist"),
    ("", ""),
])
func noteTitleTakesTheTrimmedFirstLine(input: String, expected: String) {
    #expect(noteTitle(input) == expected)
}
