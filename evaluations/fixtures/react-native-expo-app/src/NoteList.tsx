import { FlatList, StyleSheet, Text } from 'react-native';

export type Note = {
  id: string;
  title: string;
};

type NoteListProps = {
  notes: Note[];
};

export function NoteList({ notes }: NoteListProps) {
  return (
    <FlatList
      data={notes}
      keyExtractor={(note) => note.id}
      renderItem={({ item }) => <Text style={styles.note}>{item.title}</Text>}
    />
  );
}

const styles = StyleSheet.create({
  note: {
    padding: 16,
  },
});
