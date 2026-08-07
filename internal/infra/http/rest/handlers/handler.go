package handlers

type Handlers struct {
	Catalog *CatalogHandler
	Queue   *QueueHandler
}

func New(catalog *CatalogHandler, queue *QueueHandler) *Handlers {
	return &Handlers{
		Catalog: catalog,
		Queue:   queue,
	}
}
