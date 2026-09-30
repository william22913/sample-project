package endpoint

type HTTPEndpoint interface {
	RegisterEndpoint()
}

type Endpoint interface {
	AddEndpoint(controller HTTPEndpoint) *endpoint
	ServeEndpoint()
}
