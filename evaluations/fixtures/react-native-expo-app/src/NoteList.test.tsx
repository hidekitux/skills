import { render, screen } from '@testing-library/react-native';

import { NoteList } from './NoteList';

describe('NoteList', () => {
  it('shows each note title', () => {
    render(
      <NoteList
        notes={[
          { id: '1', title: 'Buy milk' },
          { id: '2', title: 'Call the dentist' },
        ]}
      />,
    );

    expect(screen.getByText('Buy milk')).toBeTruthy();
    expect(screen.getByText('Call the dentist')).toBeTruthy();
  });
});
