import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, expect, test } from "vitest";
import { NoteList } from "./NoteList";

afterEach(() => {
  cleanup();
});

test("renders each note", () => {
  render(<NoteList notes={["Buy milk", "Call the dentist"]} />);

  expect(screen.getByText("Buy milk")).toBeTruthy();
  expect(screen.getByText("Call the dentist")).toBeTruthy();
});
