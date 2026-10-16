import SwiftUI

@main
struct NotesApp: App {
    var body: some Scene {
        WindowGroup {
            NavigationStack {
                NoteListView(notes: ["Buy milk", "Call the dentist"])
                    .navigationTitle("Notes")
            }
        }
    }
}
