package handlers

type Handlers struct {
	Catalog *CatalogHandler
}

func New(catalog *CatalogHandler) *Handlers {
	return &Handlers{
		Catalog: catalog,
	}
}
