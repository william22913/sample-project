package health

func NewHealthEndpoint(
	list ...listTools,
) HealthEndpoint {
	tool := make(map[string]Tools)

	for i := 0; i < len(list); i++ {
		tool[list[i].name] = list[i].tools
	}

	return HealthEndpoint{
		tools: tool,
	}
}

func NewListTools(name string, tools Tools) listTools {
	return listTools{
		name:  name,
		tools: tools,
	}
}
