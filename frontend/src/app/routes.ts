export const ROUTES = {
  catalog: '/',
  item: (id = ':itemId') => `/item/${id}`,
  queue: (entryId = ':entryId') => `/queue/${entryId}`,
  checkout: (itemId = ':itemId') => `/checkout/${itemId}`,
  success: (orderId = ':orderId') => `/success/${orderId}`,
} as const;
