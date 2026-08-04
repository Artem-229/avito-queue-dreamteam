import { Link } from 'react-router-dom';

import { ROUTES } from '@/app/routes';

import { PagePlaceholder } from './PagePlaceholder';

export function NotFoundPage() {
  return (
    <PagePlaceholder
      title="Страница не найдена"
      description="Такого маршрута нет. Вернитесь в каталог и выберите товар."
    >
      <Link to={ROUTES.catalog}>← В каталог</Link>
    </PagePlaceholder>
  );
}
