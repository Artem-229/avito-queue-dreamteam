import { afterEach, describe, expect, it } from 'vitest';

import { getAdminKey, useAdminStore } from './model';

afterEach(() => {
  useAdminStore.getState().clear();
});

describe('admin store', () => {
  it('хранит и сбрасывает ключ администратора', () => {
    expect(getAdminKey()).toBeNull();

    useAdminStore.getState().setKey('secret');
    expect(getAdminKey()).toBe('secret');

    useAdminStore.getState().clear();
    expect(getAdminKey()).toBeNull();
  });
});
