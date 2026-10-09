export function NoteList({ notes }: { notes: string[] }) {
  return (
    <ul>
      {notes.map((note) => (
        <li key={note}>{note}</li>
      ))}
    </ul>
  );
}
