import 'package:test/test.dart';
import 'package:textkit/textkit.dart';

void main() {
  group('slug', () {
    test('joins words with hyphens', () {
      expect(slug('Hello World'), 'hello-world');
    });

    test('drops punctuation', () {
      expect(slug('Dart, Go & Rust!'), 'dart-go-rust');
    });
  });
}
