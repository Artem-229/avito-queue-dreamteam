-- +goose Up
-- +goose StatementBegin
-- Демо-каталог. Правила набора:
--   * в каждой категории минимум два товара с остатком — блок «похожие лоты»
--     в состояниях sold_out/expired не должен быть пустым (S-05);
--   * есть товар с total_stock = 1 — демонстрация конкуренции за последнюю
--     единицу (CAT-02);
--   * есть товар с total_stock = 0 — состояние sold_out видно сразу из каталога.
INSERT INTO catalog_items (name, price_kopecks, total_stock, hold_ttl_seconds, category, seller_name)
VALUES
    -- Коллекционные фигуры
    ('Бюст Дзержинского', 50000, 150, 90, 'Коллекционные фигуры', 'Ретро Сувениры'),
    ('Статуэтка «Рабочий и колхозница»', 240000, 5, 90, 'Коллекционные фигуры', 'Ретро Сувениры'),
    ('Фигурка космонавта СССР', 120000, 8, 90, 'Коллекционные фигуры', 'Барахолка №1'),

    -- Одежда
    ('Футболка I EAT CEMENT', 350000, 50, 90, 'Одежда', 'Streetwear Point'),
    ('Худи Heron Preston оверсайз', 890000, 4, 90, 'Одежда', 'Streetwear Point'),
    ('Кепка New Era 59FIFTY', 280000, 10, 90, 'Одежда', 'Cap Store'),

    -- Аниме-мерч
    ('Кружка Azumanga Daiho', 45600, 12, 90, 'Аниме-мерч', 'Anime Store'),
    ('Виниловая пластинка Nier: Automata', 99999923, 1, 90, 'Аниме-мерч', 'Vinyl Corner'),
    ('Плюш Рей', 169300, 0, 90, 'Аниме-мерч', 'Anime Store'),

    -- Музыкальные аксессуары
    ('Медиатор для гитары Jazz III', 39900, 200, 90, 'Музыкальные аксессуары', 'Music Gear'),
    ('Струны Elixir Nanoweb 10-46', 145000, 25, 90, 'Музыкальные аксессуары', 'Music Gear'),
    ('Каподастр Shubb C1', 210000, 6, 90, 'Музыкальные аксессуары', 'Guitar Corner'),

    -- Книги и артбуки
    ('Арт бук Metal Gear Solid', 1500000, 3, 90, 'Книги и артбуки', 'Geek Books'),
    ('Артбук The Art of Cyberpunk 2077', 980000, 1, 90, 'Книги и артбуки', 'Geek Books'),
    ('Артбук Dark Souls: Design Works', 750000, 7, 90, 'Книги и артбуки', 'Book Hunter');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM catalog_items
WHERE name IN (
    'Бюст Дзержинского',
    'Статуэтка «Рабочий и колхозница»',
    'Фигурка космонавта СССР',
    'Футболка I EAT CEMENT',
    'Худи Heron Preston оверсайз',
    'Кепка New Era 59FIFTY',
    'Кружка Azumanga Daiho',
    'Виниловая пластинка Nier: Automata',
    'Плюш Рей',
    'Медиатор для гитары Jazz III',
    'Струны Elixir Nanoweb 10-46',
    'Каподастр Shubb C1',
    'Арт бук Metal Gear Solid',
    'Артбук The Art of Cyberpunk 2077',
    'Артбук Dark Souls: Design Works'
);
-- +goose StatementEnd
