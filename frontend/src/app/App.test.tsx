import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { App } from './App';

describe('App', () => {
  it('рендерит каталог на стартовом маршруте', () => {
    render(<App />);

    expect(
      screen.getByRole('heading', { name: 'Дефицитные товары' }),
    ).toBeInTheDocument();
  });
});
