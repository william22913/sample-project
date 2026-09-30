package endpoint

func NewEndpoint() Endpoint {
	return &endpoint{}
}

type endpoint struct {
	endpoints []HTTPEndpoint
}

func (e *endpoint) AddEndpoint(controller HTTPEndpoint) *endpoint {
	e.endpoints = append(e.endpoints, controller)
	return e
}

func (e *endpoint) ServeEndpoint() {
	for keys := range e.endpoints {
		e.endpoints[keys].RegisterEndpoint()
	}
}
