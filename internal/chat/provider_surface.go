package chat

import (
	"errors"
	"fmt"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/google/uuid"
)

type providerSurface string

const (
	providerSurfaceEphemeralChat providerSurface = "ephemeral_chat"
	providerSurfaceHarnessAI     providerSurface = "harness_ai"
	providerSurfaceSkillAI       providerSurface = "skill_ai"
)

func chatProviderSurfaceForSource(source Source) providerSurface {
	switch source {
	case SourceHarnessEditor:
		return providerSurfaceHarnessAI
	case SourceSkillEditor:
		return providerSurfaceSkillAI
	default:
		return providerSurfaceEphemeralChat
	}
}

func resolveProviderCapabilityForSurface(
	providerItem catalogdomain.AgentProvider,
	surface providerSurface,
) catalogdomain.AgentProviderCapability {
	providerItem = catalogdomain.DeriveAgentProviderCapabilities(providerItem)
	switch surface {
	case providerSurfaceHarnessAI:
		return providerItem.Capabilities.HarnessAI
	case providerSurfaceSkillAI:
		return providerItem.Capabilities.SkillAI
	default:
		return providerItem.Capabilities.EphemeralChat
	}
}

func validateProviderForSurface(
	providerItem catalogdomain.AgentProvider,
	surface providerSurface,
	runtimeSupports func(catalogdomain.AgentProvider) bool,
) (catalogdomain.AgentProvider, error) {
	capability := resolveProviderCapabilityForSurface(providerItem, surface)
	switch capability.State {
	case catalogdomain.AgentProviderCapabilityStateUnsupported:
		return catalogdomain.AgentProvider{}, providerResolutionError(ErrProviderUnsupported, providerItem, capability)
	case catalogdomain.AgentProviderCapabilityStateUnavailable:
		return catalogdomain.AgentProvider{}, providerResolutionError(ErrProviderUnavailable, providerItem, capability)
	}
	if !runtimeSupports(providerItem) {
		return catalogdomain.AgentProvider{}, fmt.Errorf("%w: provider=%s reason=runtime_missing", ErrProviderUnsupported, providerItem.Name)
	}
	return providerItem, nil
}

func resolveProviderForSurface(
	providers []catalogdomain.AgentProvider,
	defaultProviderID *uuid.UUID,
	requestedProviderID *uuid.UUID,
	surface providerSurface,
	runtimeSupports func(catalogdomain.AgentProvider) bool,
) (catalogdomain.AgentProvider, error) {
	firstUnavailable := error(nil)
	firstUnsupported := error(nil)
	recordIssue := func(err error) {
		switch {
		case err == nil:
		case errors.Is(err, ErrProviderUnavailable) && firstUnavailable == nil:
			firstUnavailable = err
		case errors.Is(err, ErrProviderUnsupported) && firstUnsupported == nil:
			firstUnsupported = err
		}
	}

	if requestedProviderID != nil {
		providerItem, ok := findProvider(providers, *requestedProviderID)
		if !ok {
			return catalogdomain.AgentProvider{}, ErrProviderNotFound
		}
		return validateProviderForSurface(providerItem, surface, runtimeSupports)
	}

	if defaultProviderID != nil {
		if providerItem, ok := findProvider(providers, *defaultProviderID); ok {
			validated, err := validateProviderForSurface(providerItem, surface, runtimeSupports)
			if err == nil {
				return validated, nil
			}
			recordIssue(err)
		}
	}

	for _, providerItem := range providers {
		validated, err := validateProviderForSurface(providerItem, surface, runtimeSupports)
		if err == nil {
			return validated, nil
		}
		recordIssue(err)
	}

	if firstUnavailable != nil {
		return catalogdomain.AgentProvider{}, firstUnavailable
	}
	if firstUnsupported != nil {
		return catalogdomain.AgentProvider{}, firstUnsupported
	}
	return catalogdomain.AgentProvider{}, ErrProviderNotFound
}
