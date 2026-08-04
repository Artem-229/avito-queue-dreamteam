import { createBrowserRouter } from 'react-router-dom';

import { AppLayout } from '@/widgets/layout/AppLayout';
import { CatalogPage } from '@/pages/CatalogPage';
import { CheckoutPage } from '@/pages/CheckoutPage';
import { ItemPage } from '@/pages/ItemPage';
import { NotFoundPage } from '@/pages/NotFoundPage';
import { QueuePage } from '@/pages/QueuePage';
import { SuccessPage } from '@/pages/SuccessPage';

export const router = createBrowserRouter([
  {
    element: <AppLayout />,
    children: [
      { path: '/', element: <CatalogPage /> },
      { path: '/item/:itemId', element: <ItemPage /> },
      { path: '/queue/:entryId', element: <QueuePage /> },
      { path: '/checkout/:itemId', element: <CheckoutPage /> },
      { path: '/success/:orderId', element: <SuccessPage /> },
      { path: '*', element: <NotFoundPage /> },
    ],
  },
]);
