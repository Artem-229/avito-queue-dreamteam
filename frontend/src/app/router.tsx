import { createBrowserRouter } from 'react-router-dom';

import { AppLayout } from '@/widgets/layout/AppLayout';
import { CatalogPage } from '@/pages/CatalogPage';
import { CheckoutPage } from '@/pages/CheckoutPage';
import { ItemPage } from '@/pages/ItemPage';
import { MetricsPage } from '@/pages/MetricsPage';
import { NotFoundPage } from '@/pages/NotFoundPage';
import { QueuePage } from '@/pages/QueuePage';
import { SimulatorPage } from '@/pages/SimulatorPage';
import { SuccessPage } from '@/pages/SuccessPage';
import { UiKitPage } from '@/pages/UiKitPage/UiKitPage';

export const router = createBrowserRouter(
  [
    {
      element: <AppLayout />,
      children: [
        { path: '/', element: <CatalogPage /> },
        { path: '/item/:itemId', element: <ItemPage /> },
        { path: '/queue/:entryId', element: <QueuePage /> },
        { path: '/checkout/:itemId', element: <CheckoutPage /> },
        { path: '/success/:orderId', element: <SuccessPage /> },
        { path: '/ui-kit', element: <UiKitPage /> },
        { path: '/simulator', element: <SimulatorPage /> },
        { path: '/metrics', element: <MetricsPage /> },
        { path: '*', element: <NotFoundPage /> },
      ],
    },
  ],
  {
    future: {
      v7_relativeSplatPath: true,
    },
  },
);
