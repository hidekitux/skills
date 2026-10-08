package example.notes

import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp

/** Shows the notes as a scrolling list. Every platform entry point calls it. */
@Composable
fun App(notes: List<String> = emptyList(), modifier: Modifier = Modifier) {
    MaterialTheme {
        Surface(modifier = modifier.fillMaxSize()) {
            LazyColumn {
                items(notes) { note ->
                    Text(text = note, modifier = Modifier.padding(16.dp))
                }
            }
        }
    }
}
