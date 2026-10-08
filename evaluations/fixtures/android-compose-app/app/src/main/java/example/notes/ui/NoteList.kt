package example.notes.ui

import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.tooling.preview.Preview
import androidx.compose.ui.unit.dp

@Composable
fun NoteList(notes: List<String>, modifier: Modifier = Modifier) {
    LazyColumn(modifier = modifier) {
        items(notes) { note ->
            Text(text = note, modifier = Modifier.padding(16.dp))
        }
    }
}

@Preview
@Composable
private fun NoteListPreview() {
    NoteList(notes = listOf("Buy milk", "Call the dentist"))
}
