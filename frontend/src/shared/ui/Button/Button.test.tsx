import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { Button } from './Button';

describe('Button', () => {
  it('рендерит текст и реагирует на клик', () => {
    const onClick = vi.fn();
    render(<Button onClick={onClick}>Купить</Button>);

    const button = screen.getByRole('button', { name: 'Купить' });
    fireEvent.click(button);

    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it('в состоянии loading заблокирована и помечена aria-busy', () => {
    const onClick = vi.fn();
    render(
      <Button loading onClick={onClick}>
        Купить
      </Button>,
    );

    const button = screen.getByRole('button');
    fireEvent.click(button);

    expect(button).toBeDisabled();
    expect(button).toHaveAttribute('aria-busy', 'true');
    expect(onClick).not.toHaveBeenCalled();
  });
});
