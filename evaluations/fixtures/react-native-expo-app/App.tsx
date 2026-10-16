import { StatusBar } from 'expo-status-bar';
import { SafeAreaView, StyleSheet } from 'react-native';

import { NoteList } from './src/NoteList';

const notes = [
  { id: '1', title: 'Buy milk' },
  { id: '2', title: 'Call the dentist' },
];

export default function App() {
  return (
    <SafeAreaView style={styles.container}>
      <NoteList notes={notes} />
      <StatusBar style="auto" />
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
});
