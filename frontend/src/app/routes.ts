export const ROUTES = {
  catalog: '/',
  item: (id = ':itemId') => `/item/${id}`,
  queue: (itemId = ':itemId') => `/queue/${itemId}`,
  checkout: (itemId = ':itemId') => `/checkout/${itemId}`,
  success: (orderId = ':orderId') => `/success/${orderId}`,
} as const;
