import Testing
import TextKit

@Test(arguments: [
    ("Hello, World", "hello-world"),
    ("  Swift 6  ", "swift-6"),
    ("", ""),
])
func slugJoinsLetterAndDigitRuns(input: String, expected: String) {
    #expect(slug(input) == expected)
}
