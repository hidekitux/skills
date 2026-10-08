package example.notes

import androidx.compose.ui.window.Window
import androidx.compose.ui.window.application
import example.notes.resources.Res
import example.notes.resources.app_title
import org.jetbrains.compose.resources.stringResource

fun main() = application {
    Window(onCloseRequest = ::exitApplication, title = stringResource(Res.string.app_title)) {
        App()
    }
}
