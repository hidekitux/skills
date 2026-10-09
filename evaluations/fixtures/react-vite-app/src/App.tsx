import { NoteList } from "./NoteList";

const notes: string[] = ["Buy milk", "Call the dentist"];

export function App() {
  return (
    <main>
      <h1>Notes</h1>
      <NoteList notes={notes} />
    </main>
  );
}
