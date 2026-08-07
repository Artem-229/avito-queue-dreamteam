import { act, renderHook } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { getSecondsLeft, useCountdown } from './useCountdown';

describe('getSecondsLeft', () => {
  it('округляет вверх оставшиеся секунды', () => {
    expect(getSecondsLeft(10_500, 0)).toBe(11);
  });

  it('не уходит ниже нуля', () => {
    expect(getSecondsLeft(1_000, 5_000)).toBe(0);
  });
});

describe('useCountdown', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(0);
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('уменьшает счётчик и вызывает onExpire по истечении', () => {
    const onExpire = vi.fn();
    const { result } = renderHook(() =>
      useCountdown(2_000, onExpire),
    );

    expect(result.current.secondsLeft).toBe(2);
    expect(result.current.isExpired).toBe(false);

    act(() => {
      vi.advanceTimersByTime(2_000);
    });

    expect(result.current.secondsLeft).toBe(0);
    expect(result.current.isExpired).toBe(true);
    expect(onExpire).toHaveBeenCalledTimes(1);
  });
});
