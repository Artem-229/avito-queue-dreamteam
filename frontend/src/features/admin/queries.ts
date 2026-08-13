import { useMutation, useQueryClient } from '@tanstack/react-query';

import { itemKeys } from '@/entities/item';

import {
  adminCreateItem,
  adminDeleteItem,
  adminLogin,
  adminPatchItem,
  type AdminItemInput,
} from './api';
import { getAdminKey, useAdminStore } from './model';

export function useAdminLogin() {
  const setKey = useAdminStore((state) => state.setKey);

  return useMutation({
    mutationFn: async (secretKey: string) => {
      await adminLogin(secretKey);
      return secretKey;
    },
    onSuccess: (secretKey) => {
      setKey(secretKey);
    },
  });
}

export function useAdminSaveItem() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (params: { id: string | null; input: AdminItemInput }) => {
      const key = getAdminKey();
      if (!key) throw new Error('Нет ключа администратора');
      return params.id
        ? adminPatchItem(key, params.id, params.input)
        : adminCreateItem(key, params.input);
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: itemKeys.all });
    },
  });
}

export function useAdminDeleteItem() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (id: string) => {
      const key = getAdminKey();
      if (!key) throw new Error('Нет ключа администратора');
      return adminDeleteItem(key, id);
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: itemKeys.all });
    },
  });
}
