import 'package:flutter/material.dart';

class NoteList extends StatelessWidget {
  const NoteList({super.key, required this.notes});

  final List<String> notes;

  @override
  Widget build(BuildContext context) {
    return ListView.builder(
      itemCount: notes.length,
      itemBuilder: (context, index) => ListTile(title: Text(notes[index])),
    );
  }
}
