import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:notes/note_list.dart';

void main() {
  testWidgets('shows each note', (tester) async {
    await tester.pumpWidget(
      const MaterialApp(
        home: Scaffold(body: NoteList(notes: ['Buy milk', 'Call Ana'])),
      ),
    );

    expect(find.text('Buy milk'), findsOneWidget);
    expect(find.text('Call Ana'), findsOneWidget);
  });
}
