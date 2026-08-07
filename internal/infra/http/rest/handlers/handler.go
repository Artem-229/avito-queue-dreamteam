package handlers

type Handlers struct {
	Catalog  *CatalogHandler
	Checkout *CheckoutHandler
}

func New(catalog *CatalogHandler, checkout *CheckoutHandler) *Handlers {
	return &Handlers{
		Catalog:  catalog,
		Checkout: checkout,
	}
}
