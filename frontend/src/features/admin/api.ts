import { apiRequest } from '@/shared/api';

export interface AdminItemInput {
  name: string;
  price_kopecks: number;
  total_stock: number;
  hold_ttl_seconds: number;
  category: string;
  seller_name: string;
}

export function adminLogin(secretKey: string): Promise<{ success: boolean }> {
  return apiRequest<{ success: boolean }>('/api/v1/admin/login', {
    method: 'POST',
    body: JSON.stringify({ secret_key: secretKey }),
  });
}

function adminHeaders(key: string): Record<string, string> {
  return { 'X-Admin-Key': key };
}

export function adminCreateItem(
  key: string,
  input: AdminItemInput,
): Promise<unknown> {
  return apiRequest<unknown>('/api/v1/admin/catalog', {
    method: 'POST',
    headers: adminHeaders(key),
    body: JSON.stringify(input),
  });
}

export function adminPatchItem(
  key: string,
  id: string,
  input: AdminItemInput,
): Promise<unknown> {
  return apiRequest<unknown>(
    `/api/v1/admin/catalog/${encodeURIComponent(id)}`,
    {
      method: 'PATCH',
      headers: adminHeaders(key),
      body: JSON.stringify(input),
    },
  );
}

export function adminDeleteItem(key: string, id: string): Promise<void> {
  return apiRequest<void>(`/api/v1/admin/catalog/${encodeURIComponent(id)}`, {
    method: 'DELETE',
    headers: adminHeaders(key),
  });
}
