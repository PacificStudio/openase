package issueconnector

import (
	"fmt"
	"slices"

	domain "github.com/BetterAndBetterII/openase/internal/domain/issueconnector"
)

type Registry struct {
	connectors map[domain.Type]domain.Connector
}

func NewRegistry(connectors ...domain.Connector) (*Registry, error) {
	registry := &Registry{
		connectors: map[domain.Type]domain.Connector{},
	}
	for _, connector := range connectors {
		if err := registry.Register(connector); err != nil {
			return nil, err
		}
	}

	return registry, nil
}

func (r *Registry) Register(connector domain.Connector) error {
	if r == nil {
		return fmt.Errorf("connector registry is nil")
	}
	if connector == nil {
		return fmt.Errorf("connector must not be nil")
	}

	connectorType, err := domain.ParseType(connector.ID())
	if err != nil {
		return fmt.Errorf("register connector %q: %w", connector.ID(), err)
	}
	if _, exists := r.connectors[connectorType]; exists {
		return fmt.Errorf("connector %q already registered", connectorType)
	}

	r.connectors[connectorType] = connector
	return nil
}

func (r *Registry) Get(connectorType domain.Type) (domain.Connector, error) {
	if r == nil {
		return nil, fmt.Errorf("connector registry is nil")
	}

	connector, ok := r.connectors[connectorType]
	if !ok {
		return nil, fmt.Errorf("connector %q is not registered", connectorType)
	}

	return connector, nil
}

func (r *Registry) ListTypes() []domain.Type {
	if r == nil || len(r.connectors) == 0 {
		return nil
	}

	types := make([]domain.Type, 0, len(r.connectors))
	for connectorType := range r.connectors {
		types = append(types, connectorType)
	}
	slices.Sort(types)

	return types
}
