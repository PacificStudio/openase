package secrets

import (
	"fmt"
	"slices"
	"strings"

	domain "github.com/BetterAndBetterII/openase/internal/domain/secrets"
)

func BuildRuntimeEnvironment(resolved []domain.ResolvedSecret) ([]string, error) {
	if len(resolved) == 0 {
		return nil, nil
	}

	sorted := append([]domain.ResolvedSecret(nil), resolved...)
	slices.SortFunc(sorted, func(a, b domain.ResolvedSecret) int {
		return strings.Compare(a.BindingKey, b.BindingKey)
	})

	environment := make([]string, 0, len(sorted))
	for _, item := range sorted {
		key, err := domain.NormalizeName(item.BindingKey)
		if err != nil {
			return nil, fmt.Errorf("runtime binding key %q: %w", item.BindingKey, err)
		}
		environment = append(environment, key+"="+item.Value)
	}
	return environment, nil
}
