package handlers

type Handlers struct {
	Catalog  *CatalogHandler
	Checkout *CheckoutHandler
	Queue    *QueueHandler
}

func New(catalog *CatalogHandler, queue *QueueHandler, checkout *CheckoutHandler) *Handlers {
	return &Handlers{
		Catalog:  catalog,
		Checkout: checkout,
		Queue:    queue,
	}
}
