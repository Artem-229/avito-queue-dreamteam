package domain

import "github.com/google/uuid"

// ItemStats — разрез по одному товару для стенда честности.
type ItemStats struct {
	ItemID  uuid.UUID `json:"item_id"`
	Title   string    `json:"title"`
	Waiting int       `json:"waiting"`
	Granted int       `json:"granted"`
	// StockLeft — свободные слоты: тираж минус выкупленное минус удерживаемые
	// права. Именно это число отвечает на вопрос «сколько ещё можно продать
	// прямо сейчас», а total_stock на него не отвечает.
	StockLeft int `json:"stock_left"`
}

// QueueStats — сводка по всей системе (GET /api/v1/stats). Главная её задача
// на защите — показать, что granted + used никогда не превышает тираж: цифры
// берутся из тех же счётчиков, которые стережёт констрейнт
// catalog_items_stock_invariant.
type QueueStats struct {
	TotalWaiting int `json:"total_waiting"`
	TotalGranted int `json:"total_granted"`
	Purchased    int `json:"purchased"`
	Expired      int `json:"expired"`
	// Conversion — доля прав, дошедших до покупки, среди прав, у которых
	// история уже закончилась (куплено или сгорело). Права, которые ещё
	// держат, в знаменатель не входят: их судьба неизвестна, и включение
	// занижало бы конверсию тем сильнее, чем больше людей в очереди прямо
	// сейчас. nil, пока ни одно право не завершилось, — делить не на что.
	Conversion *float64 `json:"conversion"`
	// AvgWaitSeconds — среднее время от постановки в очередь до выдачи права.
	// nil, пока ни одного права не выдавали.
	AvgWaitSeconds *int `json:"avg_wait_seconds"`
	// Items — не nil, а пустой срез: в JSON null вместо массива заставил бы
	// клиента защищаться от него отдельной веткой.
	Items []ItemStats `json:"items"`
}
