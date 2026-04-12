package catalog

import "testing"

func TestBuiltinAgentProviderModelOptionsCodex(t *testing.T) {
	options := BuiltinAgentProviderModelOptions(AgentProviderAdapterTypeCodexAppServer)
	if len(options) != 8 {
		t.Fatalf("BuiltinAgentProviderModelOptions(codex) count = %d, want 8", len(options))
	}
	if options[0].ID != "gpt-5.4" || !options[0].Recommended {
		t.Fatalf("first codex option = %+v, want recommended gpt-5.4", options[0])
	}
	if options[len(options)-1].ID != "gpt-5.1-codex-mini" {
		t.Fatalf("last codex option = %+v, want gpt-5.1-codex-mini", options[len(options)-1])
	}
}

func TestBuiltinAgentProviderModelOptionsClaude(t *testing.T) {
	options := BuiltinAgentProviderModelOptions(AgentProviderAdapterTypeClaudeCodeCLI)
	if len(options) != 3 {
		t.Fatalf("BuiltinAgentProviderModelOptions(claude) count = %d, want 3", len(options))
	}
	if options[0].ID != "claude-opus-4-6" || options[0].Label != "Default" || !options[0].Recommended {
		t.Fatalf("first claude option = %+v, want recommended default opus", options[0])
	}
	if options[1].ID != "claude-sonnet-4-6" || options[2].ID != "claude-haiku-4-5" {
		t.Fatalf("claude options = %+v", options)
	}
	if options[1].Reasoning == nil || options[1].Reasoning.DefaultEffort != nil {
		t.Fatalf("claude sonnet reasoning = %+v, want plan-dependent default with nil default_effort", options[1].Reasoning)
	}
	if got := reasoningEffortStrings(options[1].Reasoning.SupportedEfforts); len(got) != 3 || got[0] != "low" || got[len(got)-1] != "high" {
		t.Fatalf("claude sonnet supported efforts = %v, want [low medium high]", got)
	}
	if options[2].Reasoning == nil || options[2].Reasoning.State != AgentProviderCapabilityStateUnsupported {
		t.Fatalf("claude haiku reasoning = %+v, want unsupported", options[2].Reasoning)
	}
}

func TestBuiltinAgentProviderModelOptionsGemini(t *testing.T) {
	options := BuiltinAgentProviderModelOptions(AgentProviderAdapterTypeGeminiCLI)
	if len(options) != 7 {
		t.Fatalf("BuiltinAgentProviderModelOptions(gemini) count = %d, want 7", len(options))
	}
	if options[0].ID != "auto-gemini-2.5" || !options[0].Recommended {
		t.Fatalf("first gemini option = %+v, want recommended auto-gemini-2.5", options[0])
	}
	if options[4].ID != "auto-gemini-3" || !options[4].Preview {
		t.Fatalf("preview gemini option = %+v, want preview auto-gemini-3", options[4])
	}
}

func TestBuiltinAgentProviderModelOptionsReturnsClone(t *testing.T) {
	options := BuiltinAgentProviderModelOptions(AgentProviderAdapterTypeCodexAppServer)
	options[0].ID = "changed"
	options[0].PricingConfig.Rates.InputPerToken = 123
	options[0].Reasoning.SupportedEfforts[0] = AgentProviderReasoningEffortMinimal

	fresh := BuiltinAgentProviderModelOptions(AgentProviderAdapterTypeCodexAppServer)
	if fresh[0].ID != "gpt-5.4" {
		t.Fatalf("BuiltinAgentProviderModelOptions() did not return a clone: %+v", fresh[0])
	}
	if fresh[0].PricingConfig == nil || fresh[0].PricingConfig.Rates.InputPerToken == 123 {
		t.Fatalf("BuiltinAgentProviderModelOptions() did not clone pricing config: %+v", fresh[0])
	}
	if fresh[0].Reasoning == nil || fresh[0].Reasoning.SupportedEfforts[0] != AgentProviderReasoningEffortLow {
		t.Fatalf("BuiltinAgentProviderModelOptions() did not clone reasoning config: %+v", fresh[0])
	}
}

func TestBuiltinAgentProviderModelOptionsUnknownAdapterReturnsNil(t *testing.T) {
	options := BuiltinAgentProviderModelOptions(AgentProviderAdapterType("unknown"))
	if options != nil {
		t.Fatalf("BuiltinAgentProviderModelOptions(unknown) = %+v, want nil", options)
	}
}

func TestBuiltinAgentProviderAdaptersWithModelOptionsReturnsClone(t *testing.T) {
	adapters := BuiltinAgentProviderAdaptersWithModelOptions()
	if len(adapters) != 3 {
		t.Fatalf("BuiltinAgentProviderAdaptersWithModelOptions() count = %d, want 3", len(adapters))
	}
	if adapters[0] != AgentProviderAdapterTypeCodexAppServer {
		t.Fatalf("first adapter = %q, want %q", adapters[0], AgentProviderAdapterTypeCodexAppServer)
	}

	adapters[0] = AgentProviderAdapterType("changed")
	fresh := BuiltinAgentProviderAdaptersWithModelOptions()
	if fresh[0] != AgentProviderAdapterTypeCodexAppServer {
		t.Fatalf("BuiltinAgentProviderAdaptersWithModelOptions() did not return a clone: %+v", fresh)
	}
}

func TestBuiltinPricingConfigPointer(t *testing.T) {
	ptr := builtinPricingConfigPointer(AgentProviderAdapterTypeCodexAppServer, "gpt-5.4")
	if ptr == nil || ptr.ModelID != "gpt-5.4" {
		t.Fatalf("builtinPricingConfigPointer() = %+v, want gpt-5.4 pricing config", ptr)
	}
	if builtinPricingConfigPointer(AgentProviderAdapterTypeCodexAppServer, "unknown") != nil {
		t.Fatal("builtinPricingConfigPointer() expected nil for unknown model")
	}
}
