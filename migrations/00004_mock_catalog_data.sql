-- +goose Up
-- +goose StatementBegin
INSERT INTO catalog_items (name, price, total_stock, category, seller_name)
VALUES
    ('Бюст Дзержинского', 500.00, 150, 'Коллекционные фигуры', 'Ретро Сувениры'),
    ('Футболка I EAT CEMENT', 3500.00, 50, 'Одежда', 'Streetwear Point'),
    ('Кружка Azumanga Daiho', 456.00, 12, 'Аниме-мерч', 'Anime Store'),
    ('Медиатор для гитары Jazz III', 399.00, 200, 'Музыкальные аксессуары', 'Music Gear'),
    ('Арт бук Metal Gear Solid', 15000.00, 3, 'Книги и артбуки', 'Geek Books'),
    ('Виниловая пластинка Nier: Automata', 999999.23, 1, 'Аниме-мерч', 'Vinyl Corner'),
    ('Плюш Рей', 1693.00, 0, 'Аниме-мерч', 'Anime Store');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM catalog_items 
WHERE name IN (
    'Бюст Дзержинского', 
    'Футболка I EAT CEMENT', 
    'Кружка Azumanga Daiho', 
    'Медиатор для гитары Jazz III', 
    'Арт бук Metal Gear Solid',
    'Виниловая пластинка Nier: Automata',
    'Плюш Рей'
);
-- +goose StatementEnd