package services

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"avito-queue/internal/domain"
	"avito-queue/internal/infra/postgres/repository"
)

// fakeEmbedder заменяет GigaChat: в тестах не должно быть ни сети, ни токенов,
// ни платы за вызовы. Вектор детерминированный — от него зависит только
// порядок похожих лотов, и его задаёт сам тест.
type fakeEmbedder struct {
	mu          sync.Mutex
	vector      []float32
	err         error
	calls       []string
	invalidated []string
}

// embedderAxis раздаёт каждому эмбеддеру собственную ось. Одинаковые векторы
// у товаров из разных тестов (а прогрев проходит по всей базе) сделали бы
// порядок похожих лотов делом случая, а не тем, что проверяет тест.
var embedderAxis atomic.Int64

func newFakeEmbedder() *fakeEmbedder {
	vector := make([]float32, embeddingDim)
	vector[embedderAxis.Add(1)%embeddingDim] = 1

	return &fakeEmbedder{vector: vector}
}

func (f *fakeEmbedder) CreateEmbedding(_ context.Context, text string) ([]float32, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.calls = append(f.calls, text)
	if f.err != nil {
		return nil, f.err
	}

	return f.vector, nil
}

func (f *fakeEmbedder) Invalidate(text string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.invalidated = append(f.invalidated, text)
}

func (f *fakeEmbedder) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return len(f.calls)
}

// embeddingDim — размерность вектора в схеме (vector(1024) из миграции 00005).
// Вектор другой длины Postgres не примет.
const embeddingDim = 1024

func newTestCatalogService(pool *pgxpool.Pool, embedder Embedder) *CatalogService {
	queueRepo := repository.NewQueueEntryRepo(pool)

	return NewCatalogService(repository.NewCatalogRepository(pool), embedder, queueRepo, NewChanceCalculator(queueRepo))
}

// TestGetCatalogItem_ShowsQueueAndChance — карточка товара отвечает на вопрос
// «стоит ли вообще вставать»: сколько человек уже ждёт и какой шанс у того,
// кто встанет сейчас.
func TestGetCatalogItem_ShowsQueueAndChance(t *testing.T) {
	t.Parallel()

	pool := testDBPool(t)
	itemID := insertTestItem(t, pool, 2, 120)
	service := newTestCatalogService(pool, newFakeEmbedder())
	queue := newTestQueueService(pool)
	ctx := context.Background()

	card, err := service.GetCatalogItem(ctx, itemID)
	require.NoError(t, err)
	require.Zero(t, card.QueueSize)
	require.NotNil(t, card.ChanceIfJoin, "пока товар есть, шанс должен быть посчитан")
	require.Equal(t, 100, card.ChanceIfJoin.Percent,
		"на пустой очереди при двух единицах встающий получит право сразу")

	for range 3 {
		_, err := queue.Entry(ctx, uuid.New(), itemID)
		require.NoError(t, err)
	}

	card, err = service.GetCatalogItem(ctx, itemID)
	require.NoError(t, err)
	require.Equal(t, 1, card.QueueSize, "двое получили право, третий ждёт")
	require.NotNil(t, card.ChanceIfJoin)
	require.Contains(t, []string{"item", "global", "default"}, card.ChanceIfJoin.Basis,
		"клиент должен знать, на чём посчитан шанс: измеренная оценка и оценка по умолчанию — разные вещи")
	require.GreaterOrEqual(t, card.ChanceIfJoin.Percent, 0)
	require.LessOrEqual(t, card.ChanceIfJoin.Percent, 100)
	// Само число зависит от конверсии, накопленной всей базой, поэтому
	// арифметика шанса проверяется отдельно и без БД — см. chance_test.go.
}

// TestGetCatalogItem_NoChanceWhenSoldOut — на распроданном товаре шанс не
// показывается вовсе: ноль процентов и «шанса нет» — разные сообщения.
func TestGetCatalogItem_NoChanceWhenSoldOut(t *testing.T) {
	t.Parallel()

	pool := testDBPool(t)
	itemID := insertTestItem(t, pool, 1, 120)
	service := newTestCatalogService(pool, newFakeEmbedder())
	queue := newTestQueueService(pool)
	ctx := context.Background()

	buyer := uuid.New()
	_, err := queue.Entry(ctx, buyer, itemID)
	require.NoError(t, err)
	require.NoError(t, queue.Buy(ctx, buyer, itemID))

	card, err := service.GetCatalogItem(ctx, itemID)
	require.NoError(t, err)
	require.Nil(t, card.ChanceIfJoin)
}

func TestGetCatalogItem_UnknownItem(t *testing.T) {
	t.Parallel()

	pool := testDBPool(t)
	service := newTestCatalogService(pool, newFakeEmbedder())

	_, err := service.GetCatalogItem(context.Background(), uuid.New())
	require.ErrorIs(t, err, domain.ErrNoItemFound)
}

// TestCreateItem_StoresEmbeddingOnce — вектор считается при создании товара, а
// не при первом показе похожих лотов: наплыв не должен упираться в поход к
// LLM.
func TestCreateItem_StoresEmbeddingOnce(t *testing.T) {
	t.Parallel()

	pool := testDBPool(t)
	embedder := newFakeEmbedder()
	service := newTestCatalogService(pool, embedder)
	ctx := context.Background()

	item, err := service.CreateItem(ctx, domain.CatalogItemCreateDTO{
		Name:           "тестовый лот " + uuid.NewString(),
		PriceKopecks:   12345,
		TotalStock:     2,
		HoldTTLSeconds: 90,
		Category:       "test-category",
		SellerName:     "test-seller",
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM catalog_items WHERE id = $1`, item.ID)
	})

	require.True(t, item.HasEmbedding)
	require.Equal(t, 1, embedder.callCount(), "на создание товара — ровно один вызов эмбеддера")

	stored, err := repository.NewCatalogRepository(pool).GetItemByID(ctx, item.ID)
	require.NoError(t, err)
	require.True(t, stored.HasEmbedding, "вектор обязан лечь в базу вместе с товаром")
	require.Equal(t, item.Name, stored.Name)
}

// TestCreateItem_SurvivesEmbedderFailure — товар важнее вектора: если LLM
// недоступна, каталог обязан продолжать работать, просто без похожих лотов.
func TestCreateItem_SurvivesEmbedderFailure(t *testing.T) {
	t.Parallel()

	pool := testDBPool(t)
	embedder := newFakeEmbedder()
	embedder.err = errors.New("gigachat unavailable")
	service := newTestCatalogService(pool, embedder)

	item, err := service.CreateItem(context.Background(), domain.CatalogItemCreateDTO{
		Name:           "лот без вектора " + uuid.NewString(),
		PriceKopecks:   500,
		TotalStock:     1,
		HoldTTLSeconds: 60,
		Category:       "test-category",
		SellerName:     "test-seller",
	})
	require.NoError(t, err, "недоступность эмбеддера не должна ронять создание товара")
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM catalog_items WHERE id = $1`, item.ID)
	})

	require.False(t, item.HasEmbedding)
}

// TestPatchItem_RecomputesVectorOnlyWhenTextChanges — вектор зависит от
// названия и категории. Правка цены или остатка не повод платить за новый
// вызов LLM.
func TestPatchItem_RecomputesVectorOnlyWhenTextChanges(t *testing.T) {
	t.Parallel()

	pool := testDBPool(t)
	embedder := newFakeEmbedder()
	service := newTestCatalogService(pool, embedder)
	itemID := insertTestItem(t, pool, 5, 120)
	ctx := context.Background()

	newStock := 7
	updated, err := service.PatchItem(ctx, itemID, domain.CatalogItemPatchDTO{TotalStock: &newStock})
	require.NoError(t, err)
	require.Equal(t, 7, updated.TotalStock)
	require.Zero(t, embedder.callCount(), "правка остатка не меняет текст товара — вектор пересчитывать незачем")

	newName := "переименованный лот " + uuid.NewString()
	updated, err = service.PatchItem(ctx, itemID, domain.CatalogItemPatchDTO{Name: &newName})
	require.NoError(t, err)
	require.Equal(t, newName, updated.Name)
	require.Equal(t, 1, embedder.callCount(), "смена названия обязана пересчитать вектор")
	require.Len(t, embedder.invalidated, 1, "старый текст обязан выпасть из кэша векторов")
}

// TestPatchItem_ChangesHoldTTL — окно на оплату настраивается на каждый товар:
// это единственная настройка очереди, доступная админке.
func TestPatchItem_ChangesHoldTTL(t *testing.T) {
	t.Parallel()

	pool := testDBPool(t)
	service := newTestCatalogService(pool, newFakeEmbedder())
	queue := newTestQueueService(pool)
	itemID := insertTestItem(t, pool, 1, 120)
	ctx := context.Background()

	newTTL := 30
	_, err := service.PatchItem(ctx, itemID, domain.CatalogItemPatchDTO{HoldTTLSeconds: &newTTL})
	require.NoError(t, err)

	resp, err := queue.Entry(ctx, uuid.New(), itemID)
	require.NoError(t, err)
	require.Equal(t, domain.QueueEntryStatusGranted, resp.Status)
	require.NotNil(t, resp.ExpiresAt)
	require.InDelta(t, float64(newTTL), resp.ExpiresAt.Sub(resp.ServerTime).Seconds(), 5,
		"срок права обязан считаться по настроенному окну товара")
}

// TestDeleteItem_HidesFromCatalog — удаление мягкое: товар исчезает из выдачи,
// но история участий и статистика по нему остаются.
func TestDeleteItem_HidesFromCatalog(t *testing.T) {
	t.Parallel()

	pool := testDBPool(t)
	service := newTestCatalogService(pool, newFakeEmbedder())
	itemID := insertTestItem(t, pool, 1, 120)
	ctx := context.Background()

	require.NoError(t, service.DeleteItem(ctx, itemID))

	_, err := service.GetCatalogItem(ctx, itemID)
	require.ErrorIs(t, err, domain.ErrNoItemFound)

	var rows int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT count(*) FROM catalog_items WHERE id = $1 AND deleted_at IS NOT NULL`, itemID).Scan(&rows))
	require.Equal(t, 1, rows, "строка товара обязана остаться в базе с отметкой удаления")

	require.ErrorIs(t, service.DeleteItem(ctx, uuid.New()), domain.ErrNoItemFound)
}

// TestGetSimilarItems_SkipsSelfAndSoldOut — распроданное взамен распроданного
// не предлагается: это и есть ответ на вопрос кейса, почему пользователь
// остаётся на площадке.
func TestGetSimilarItems_SkipsSelfAndSoldOut(t *testing.T) {
	t.Parallel()

	pool := testDBPool(t)
	embedder := newFakeEmbedder()
	service := newTestCatalogService(pool, embedder)
	catalogRepo := repository.NewCatalogRepository(pool)
	queue := newTestQueueService(pool)
	ctx := context.Background()

	sourceID := insertTestItem(t, pool, 1, 120)
	aliveID := insertTestItem(t, pool, 3, 120)
	soldOutID := insertTestItem(t, pool, 1, 120)

	for _, id := range []uuid.UUID{sourceID, aliveID, soldOutID} {
		require.NoError(t, catalogRepo.SaveEmbedding(ctx, id, embedder.vector))
	}

	buyer := uuid.New()
	_, err := queue.Entry(ctx, buyer, soldOutID)
	require.NoError(t, err)
	require.NoError(t, queue.Buy(ctx, buyer, soldOutID))

	similar, err := service.GetSimilarItems(ctx, sourceID)
	require.NoError(t, err)

	ids := make(map[uuid.UUID]bool, len(similar))
	for _, item := range similar {
		ids[item.ID] = true
	}

	require.False(t, ids[sourceID], "товар не может быть похож сам на себя")
	require.False(t, ids[soldOutID], "распроданный лот предлагать нельзя")
	require.True(t, ids[aliveID], "лот с остатком обязан попасть в подборку")
	require.Zero(t, embedder.callCount(),
		"у товара уже есть вектор — похожие лоты обязаны подбираться без похода в LLM")
}

// TestWarmupEmbeddings_FillsMissingVectors — прогрев на старте: векторы
// считаются до наплыва, а не в момент, когда всем ожидающим разом показывают
// экран окончания ожидания.
func TestWarmupEmbeddings_FillsMissingVectors(t *testing.T) {
	pool := testDBPool(t)
	embedder := newFakeEmbedder()
	service := newTestCatalogService(pool, embedder)
	itemID := insertTestItem(t, pool, 1, 120)
	ctx := context.Background()

	item, err := repository.NewCatalogRepository(pool).GetItemByID(ctx, itemID)
	require.NoError(t, err)
	require.False(t, item.HasEmbedding)

	service.WarmupEmbeddings(ctx)

	item, err = repository.NewCatalogRepository(pool).GetItemByID(ctx, itemID)
	require.NoError(t, err)
	require.True(t, item.HasEmbedding, "после прогрева у товара обязан появиться вектор")
	require.NotZero(t, embedder.callCount())
}
