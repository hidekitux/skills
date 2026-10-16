import SwiftUI

struct NoteListView: View {
    let notes: [String]

    var body: some View {
        List(notes, id: \.self) { Text($0) }
    }
}

#Preview {
    NoteListView(notes: ["Buy milk", "Call the dentist"])
}
