-- +goose Up
-- +goose StatementBegin
INSERT INTO catalog_items (name, price, total_stock) 
VALUES 
    ('Бюст Дзержинского', 500.00, 150),
    ('Футболка I EAT CEMENT', 3500.00, 50),
    ('Кружка Azumanga Daiho', 456.00, 12),
    ('Медиатор для гитары Jazz III', 399.00, 200),
    ('Арт бук Metal Gear Solid', 15000.00, 3),
    ('Виниловая пластинка Nier: Automata', 999999.23, 1),
    ('Плюш Рей', 1693.00, 0);
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