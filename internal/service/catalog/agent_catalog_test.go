package catalog

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	transportinfra "github.com/BetterAndBetterII/openase/internal/infra/machinetransport"
	"github.com/google/uuid"
)

func TestCreateAgentProviderAutoDetectsCLICommand(t *testing.T) {
	repo := &stubRepository{}
	svc := New(repo, stubExecutableResolver{
		paths: map[string]string{"codex": "/usr/local/bin/codex"},
	}, nil)

	item, err := svc.CreateAgentProvider(context.Background(), domain.CreateAgentProvider{
		OrganizationID: uuid.New(),
		MachineID:      uuid.New(),
		Name:           "Codex",
		AdapterType:    domain.AgentProviderAdapterTypeCodexAppServer,
		ModelName:      "gpt-5.4",
		AuthConfig:     map[string]any{},
	})
	if err != nil {
		t.Fatalf("CreateAgentProvider returned error: %v", err)
	}
	if item.CliCommand != "/usr/local/bin/codex" {
		t.Fatalf("expected auto-detected cli command, got %+v", item)
	}
	if repo.createdProvider == nil || repo.createdProvider.CliCommand != "/usr/local/bin/codex" {
		t.Fatalf("expected repo to receive resolved cli command, got %+v", repo.createdProvider)
	}
	if repo.createdProvider == nil || repo.createdProvider.PermissionProfile != domain.AgentProviderPermissionProfileUnrestricted {
		t.Fatalf("expected default permission profile to be unrestricted, got %+v", repo.createdProvider)
	}
	if want := []string{"app-server", "--listen", "stdio://"}; !equalStrings(item.CliArgs, want) {
		t.Fatalf("expected default codex cli args %v, got %v", want, item.CliArgs)
	}
}

func TestCreateAgentProviderRejectsMissingCustomCLICommand(t *testing.T) {
	svc := New(&stubRepository{}, stubExecutableResolver{}, nil)

	_, err := svc.CreateAgentProvider(context.Background(), domain.CreateAgentProvider{
		OrganizationID: uuid.New(),
		MachineID:      uuid.New(),
		Name:           "Custom",
		AdapterType:    domain.AgentProviderAdapterTypeCustom,
		ModelName:      "manual",
		AuthConfig:     map[string]any{},
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input error, got %v", err)
	}
}

func TestCreateAgentProviderRejectsMissingExecutable(t *testing.T) {
	svc := New(&stubRepository{}, stubExecutableResolver{}, nil)

	_, err := svc.CreateAgentProvider(context.Background(), domain.CreateAgentProvider{
		OrganizationID: uuid.New(),
		MachineID:      uuid.New(),
		Name:           "Gemini",
		AdapterType:    domain.AgentProviderAdapterTypeGeminiCLI,
		ModelName:      "gemini-2.5-pro",
		AuthConfig:     map[string]any{},
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input error, got %v", err)
	}
}

func TestUpdateAgentProviderDefaultsCodexCLIArgs(t *testing.T) {
	repo := &stubRepository{
		provider: domain.AgentProvider{
			ID:             uuid.New(),
			OrganizationID: uuid.New(),
			MachineID:      uuid.New(),
			Name:           "Codex",
			AdapterType:    domain.AgentProviderAdapterTypeCodexAppServer,
			CliCommand:     "/usr/local/bin/codex",
			ModelName:      "gpt-5.3-codex",
			AuthConfig:     map[string]any{},
		},
	}
	svc := New(repo, stubExecutableResolver{}, nil)

	item, err := svc.UpdateAgentProvider(context.Background(), domain.UpdateAgentProvider{
		ID:             repo.provider.ID,
		OrganizationID: repo.provider.OrganizationID,
		MachineID:      repo.provider.MachineID,
		Name:           repo.provider.Name,
		AdapterType:    repo.provider.AdapterType,
		CliCommand:     repo.provider.CliCommand,
		ModelName:      repo.provider.ModelName,
		AuthConfig:     map[string]any{},
	})
	if err != nil {
		t.Fatalf("UpdateAgentProvider returned error: %v", err)
	}
	if want := []string{"app-server", "--listen", "stdio://"}; !equalStrings(item.CliArgs, want) {
		t.Fatalf("expected default codex cli args %v, got %v", want, item.CliArgs)
	}
	if repo.updatedProvider == nil || !equalStrings(repo.updatedProvider.CliArgs, []string{"app-server", "--listen", "stdio://"}) {
		t.Fatalf("expected repo update to receive default codex args, got %+v", repo.updatedProvider)
	}
}

func TestCreateAgentProviderNormalizesManagedPermissionFlags(t *testing.T) {
	repo := &stubRepository{}
	svc := New(repo, stubExecutableResolver{
		paths: map[string]string{
			"codex":  "/usr/local/bin/codex",
			"claude": "/usr/local/bin/claude",
			"gemini": "/usr/local/bin/gemini",
		},
	}, nil)

	cases := []struct {
		name        string
		adapterType domain.AgentProviderAdapterType
		inputArgs   []string
		wantArgs    []string
	}{
		{
			name:        "codex strips launch and permission flags",
			adapterType: domain.AgentProviderAdapterTypeCodexAppServer,
			inputArgs:   []string{"app-server", "--listen", "ws://127.0.0.1:7777", "--sandbox", "danger-full-access", "--ask-for-approval", "never", "--trace"},
			wantArgs:    []string{"app-server", "--listen", "stdio://", "--trace"},
		},
		{
			name:        "claude strips permission flags",
			adapterType: domain.AgentProviderAdapterTypeClaudeCodeCLI,
			inputArgs:   []string{"--dangerously-skip-permissions", "--permission-mode", "bypassPermissions", "--verbose"},
			wantArgs:    []string{"--verbose"},
		},
		{
			name:        "gemini strips yolo flags",
			adapterType: domain.AgentProviderAdapterTypeGeminiCLI,
			inputArgs:   []string{"--approval-mode=yolo", "--sandbox=false"},
			wantArgs:    []string{"--sandbox=false"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo.createdProvider = nil
			_, err := svc.CreateAgentProvider(context.Background(), domain.CreateAgentProvider{
				OrganizationID:    uuid.New(),
				MachineID:         uuid.New(),
				Name:              "Provider",
				AdapterType:       tc.adapterType,
				PermissionProfile: domain.AgentProviderPermissionProfileStandard,
				ModelName:         "model",
				AuthConfig:        map[string]any{},
				CliArgs:           tc.inputArgs,
			})
			if err != nil {
				t.Fatalf("CreateAgentProvider returned error: %v", err)
			}
			if repo.createdProvider == nil || !equalStrings(repo.createdProvider.CliArgs, tc.wantArgs) {
				t.Fatalf("stored cli args = %+v, want %v", repo.createdProvider, tc.wantArgs)
			}
		})
	}
}

func TestListAgentProvidersAnnotatesAvailability(t *testing.T) {
	orgID := uuid.New()
	checkedAt := time.Now().UTC().Add(-5 * time.Minute)
	repo := &stubRepository{
		listedProviders: []domain.AgentProvider{
			{
				ID:               uuid.New(),
				OrganizationID:   orgID,
				MachineID:        uuid.New(),
				MachineHost:      domain.LocalMachineHost,
				MachineStatus:    domain.MachineStatusOnline,
				MachineResources: providerAvailabilityResources(checkedAt, "claude_code", false, domain.MachineAgentAuthStatusUnknown, domain.MachineAgentAuthModeUnknown, false),
				Name:             "Claude Code",
				AdapterType:      domain.AgentProviderAdapterTypeClaudeCodeCLI,
				CliCommand:       "claude",
				ModelName:        "claude-opus-4-6",
			},
			{
				ID:               uuid.New(),
				OrganizationID:   orgID,
				MachineID:        uuid.New(),
				MachineHost:      domain.LocalMachineHost,
				MachineStatus:    domain.MachineStatusOnline,
				MachineResources: providerAvailabilityResources(checkedAt, "codex", true, domain.MachineAgentAuthStatusLoggedIn, domain.MachineAgentAuthModeLogin, true),
				Name:             "OpenAI Codex",
				AdapterType:      domain.AgentProviderAdapterTypeCodexAppServer,
				CliCommand:       "codex",
				ModelName:        "gpt-5.4",
			},
		},
	}
	svc := New(repo, stubExecutableResolver{}, nil)

	items, err := svc.ListAgentProviders(context.Background(), orgID)
	if err != nil {
		t.Fatalf("ListAgentProviders returned error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 providers, got %+v", items)
	}
	if items[0].Available {
		t.Fatalf("expected claude provider to be unavailable, got %+v", items[0])
	}
	if items[0].AvailabilityState != domain.AgentProviderAvailabilityStateUnavailable {
		t.Fatalf("expected claude provider state unavailable, got %+v", items[0])
	}
	if !items[1].Available {
		t.Fatalf("expected codex provider to be available, got %+v", items[1])
	}
	if items[1].AvailabilityState != domain.AgentProviderAvailabilityStateAvailable {
		t.Fatalf("expected codex provider state available, got %+v", items[1])
	}
	if items[1].AvailabilityCheckedAt == nil {
		t.Fatalf("expected codex provider to include availability_checked_at, got %+v", items[1])
	}
}

func TestCreateOrganizationSetsDefaultProviderToPreferredAvailableBuiltin(t *testing.T) {
	orgID := uuid.New()
	checkedAt := time.Now().UTC().Add(-5 * time.Minute)
	repo := &stubRepository{
		createdOrganization: domain.Organization{
			ID:   orgID,
			Name: "Acme",
			Slug: "acme",
		},
		listedProviders: []domain.AgentProvider{
			{
				ID:               uuid.New(),
				OrganizationID:   orgID,
				MachineID:        uuid.New(),
				MachineHost:      domain.LocalMachineHost,
				MachineStatus:    domain.MachineStatusOnline,
				MachineResources: providerAvailabilityResources(checkedAt, "claude_code", false, domain.MachineAgentAuthStatusUnknown, domain.MachineAgentAuthModeUnknown, false),
				Name:             "Claude Code",
				AdapterType:      domain.AgentProviderAdapterTypeClaudeCodeCLI,
				CliCommand:       "claude",
				ModelName:        "claude-opus-4-6",
			},
			{
				ID:               uuid.New(),
				OrganizationID:   orgID,
				MachineID:        uuid.New(),
				MachineHost:      domain.LocalMachineHost,
				MachineStatus:    domain.MachineStatusOnline,
				MachineResources: providerAvailabilityResources(checkedAt, "codex", true, domain.MachineAgentAuthStatusLoggedIn, domain.MachineAgentAuthModeLogin, true),
				Name:             "OpenAI Codex",
				AdapterType:      domain.AgentProviderAdapterTypeCodexAppServer,
				CliCommand:       "codex",
				ModelName:        "gpt-5.4",
			},
		},
	}
	svc := New(repo, stubExecutableResolver{}, nil)

	item, err := svc.CreateOrganization(context.Background(), domain.CreateOrganization{
		Name: "Acme",
		Slug: "acme",
	})
	if err != nil {
		t.Fatalf("CreateOrganization returned error: %v", err)
	}
	if item.DefaultAgentProviderID == nil {
		t.Fatalf("expected default provider to be set, got %+v", item)
	}
	if repo.updatedOrganization == nil || repo.updatedOrganization.DefaultAgentProviderID == nil {
		t.Fatalf("expected organization update with default provider, got %+v", repo.updatedOrganization)
	}
	if *item.DefaultAgentProviderID != *repo.updatedOrganization.DefaultAgentProviderID {
		t.Fatalf("expected returned org default %s to match repo update %s", item.DefaultAgentProviderID, repo.updatedOrganization.DefaultAgentProviderID)
	}
}

func TestCreateProjectSeedsDefaultStatuses(t *testing.T) {
	projectID := uuid.New()
	repo := &stubRepository{
		createdProject: domain.Project{
			ID:             projectID,
			OrganizationID: uuid.New(),
			Name:           "OpenASE",
			Slug:           "openase",
			Status:         "In Progress",
		},
	}
	resetter := &stubProjectStatusBootstrapper{}
	svc := New(repo, stubExecutableResolver{}, nil, WithProjectStatusBootstrapper(resetter))

	item, err := svc.CreateProject(context.Background(), domain.CreateProject{
		OrganizationID: repo.createdProject.OrganizationID,
		Name:           "OpenASE",
		Slug:           "openase",
		Status:         "In Progress",
	})
	if err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}
	if item.ID != projectID {
		t.Fatalf("expected created project %s, got %+v", projectID, item)
	}
	if resetter.projectID != projectID {
		t.Fatalf("expected default status bootstrap for project %s, got %s", projectID, resetter.projectID)
	}
}

func TestArchiveOrganizationDelegatesToRepository(t *testing.T) {
	orgID := uuid.New()
	repo := &stubRepository{
		createdOrganization: domain.Organization{
			ID:     orgID,
			Name:   "Acme",
			Slug:   "acme",
			Status: "archived",
		},
	}
	svc := New(repo, stubExecutableResolver{}, nil)

	item, err := svc.ArchiveOrganization(context.Background(), orgID)
	if err != nil {
		t.Fatalf("ArchiveOrganization returned error: %v", err)
	}
	if repo.archivedOrganizationID != orgID {
		t.Fatalf("expected archive organization call for %s, got %s", orgID, repo.archivedOrganizationID)
	}
	if item.Status != "archived" {
		t.Fatalf("expected archived organization, got %+v", item)
	}
}

func TestTestMachineConnectionPreservesExistingResourceSnapshot(t *testing.T) {
	machineID := uuid.New()
	orgID := uuid.New()
	checkedAt := time.Now().UTC()
	repo := &stubRepository{
		machine: domain.Machine{
			ID:             machineID,
			OrganizationID: orgID,
			Name:           "gpu-01",
			Host:           "10.0.0.8",
			Port:           22,
			Status:         domain.MachineStatusOnline,
			Resources: map[string]any{
				"transport":         "ssh",
				"cpu_usage_percent": 61.2,
				"memory_total_gb":   64.0,
				"memory_used_gb":    18.5,
				"disk_available_gb": 220.0,
				"gpu_dispatchable":  true,
				"checked_at":        checkedAt.Add(-10 * time.Minute).Format(time.RFC3339),
				"last_success":      true,
				"monitor":           map[string]any{"l1": map[string]any{"reachable": true}},
			},
		},
	}
	tester := stubMachineTester{
		probe: domain.MachineProbe{
			CheckedAt:       checkedAt,
			Transport:       "ssh",
			Output:          "probe-ok",
			DetectedOS:      domain.MachineDetectedOSLinux,
			DetectedArch:    domain.MachineDetectedArchAMD64,
			DetectionStatus: domain.MachineDetectionStatusOK,
			Resources: map[string]any{
				"transport":    "ssh",
				"host":         "10.0.0.8",
				"port":         22,
				"checked_at":   checkedAt.Format(time.RFC3339),
				"last_success": true,
			},
		},
	}
	svc := New(repo, stubExecutableResolver{}, tester)

	updated, probe, err := svc.TestMachineConnection(context.Background(), machineID)
	if err != nil {
		t.Fatalf("TestMachineConnection returned error: %v", err)
	}
	if probe.Transport != "ssh" {
		t.Fatalf("expected ssh probe transport, got %+v", probe)
	}
	if repo.recordedMachineProbe == nil {
		t.Fatal("expected machine probe to be recorded")
	}
	if got := repo.recordedMachineProbe.Resources["cpu_usage_percent"]; got != 61.2 {
		t.Fatalf("expected cpu usage snapshot to survive probe, got %+v", repo.recordedMachineProbe.Resources)
	}
	if got := repo.recordedMachineProbe.Resources["memory_used_gb"]; got != 18.5 {
		t.Fatalf("expected memory snapshot to survive probe, got %+v", repo.recordedMachineProbe.Resources)
	}
	connectionTest, ok := repo.recordedMachineProbe.Resources["connection_test"].(map[string]any)
	if !ok {
		t.Fatalf("expected connection_test payload, got %+v", repo.recordedMachineProbe.Resources)
	}
	if connectionTest["last_success"] != true {
		t.Fatalf("expected successful connection_test payload, got %+v", connectionTest)
	}
	if repo.recordedMachineProbe.DetectedOS != domain.MachineDetectedOSLinux ||
		repo.recordedMachineProbe.DetectedArch != domain.MachineDetectedArchAMD64 ||
		repo.recordedMachineProbe.DetectionStatus != domain.MachineDetectionStatusOK {
		t.Fatalf("expected detection metadata to be recorded, got %+v", repo.recordedMachineProbe)
	}
	if updated.Resources["cpu_usage_percent"] != 61.2 {
		t.Fatalf("expected returned machine to retain cpu snapshot, got %+v", updated.Resources)
	}
	if updated.Resources["connection_test"] == nil {
		t.Fatalf("expected returned machine to expose connection_test, got %+v", updated.Resources)
	}
}

func TestRefreshMachineHealthCollectsAndPersistsMultiLevelSnapshot(t *testing.T) {
	machineID := uuid.New()
	orgID := uuid.New()
	checkedAt := time.Date(2026, 3, 30, 14, 24, 24, 0, time.UTC)
	repo := &stubRepository{
		machine: domain.Machine{
			ID:             machineID,
			OrganizationID: orgID,
			Name:           "local",
			Host:           domain.LocalMachineHost,
			Port:           22,
			Status:         domain.MachineStatusOnline,
			Resources: map[string]any{
				"monitor": map[string]any{
					"l1": map[string]any{
						"checked_at":           checkedAt.Add(-time.Hour).Format(time.RFC3339),
						"consecutive_failures": 2,
					},
				},
			},
		},
	}
	collector := stubMachineHealthCollector{
		reachability: domain.MachineReachability{
			CheckedAt: checkedAt,
			Transport: "local",
			Reachable: true,
		},
		systemResources: domain.MachineSystemResources{
			CollectedAt:            checkedAt,
			CPUCores:               24,
			CPUUsagePercent:        13.5,
			MemoryTotalGB:          64,
			MemoryUsedGB:           21,
			MemoryAvailableGB:      43,
			MemoryAvailablePercent: 67.2,
			DiskTotalGB:            1024,
			DiskAvailableGB:        900,
		},
		gpuResources: domain.MachineGPUResources{
			CollectedAt: checkedAt,
			Available:   false,
		},
		agentEnvironment: domain.MachineAgentEnvironment{
			CollectedAt:  checkedAt,
			Dispatchable: true,
			CLIs: []domain.MachineAgentCLI{
				{Name: "claude_code", Installed: true, Version: "2.1.87", AuthStatus: domain.MachineAgentAuthStatusLoggedIn, AuthMode: domain.MachineAgentAuthModeLogin, Ready: true},
				{Name: "codex", Installed: true, Version: "0.117.0", AuthStatus: domain.MachineAgentAuthStatusLoggedIn, AuthMode: domain.MachineAgentAuthModeLogin, Ready: true},
				{Name: "gemini", Installed: true, Version: "0.35.2", AuthStatus: domain.MachineAgentAuthStatusUnknown, AuthMode: domain.MachineAgentAuthModeUnknown, Ready: true},
			},
		},
		fullAudit: domain.MachineFullAudit{
			CollectedAt: checkedAt,
			Git: domain.MachineGitAudit{
				Installed: true,
				UserName:  "Codex",
				UserEmail: "codex@openai.com",
			},
			GitHubCLI: domain.MachineGitHubCLIAudit{
				Installed:  true,
				AuthStatus: domain.MachineAgentAuthStatusLoggedIn,
			},
			Network: domain.MachineNetworkAudit{
				GitHubReachable: true,
				PyPIReachable:   true,
				NPMReachable:    true,
			},
		},
	}
	svc := New(repo, stubExecutableResolver{}, nil, WithMachineHealthCollector(collector))

	updated, err := svc.RefreshMachineHealth(context.Background(), machineID)
	if err != nil {
		t.Fatalf("RefreshMachineHealth returned error: %v", err)
	}
	if repo.recordedMachineProbe == nil {
		t.Fatal("expected refreshed machine resources to be persisted")
	}
	if updated.Status != domain.MachineStatusOnline {
		t.Fatalf("expected online machine after successful refresh, got %+v", updated)
	}

	monitor, ok := updated.Resources["monitor"].(map[string]any)
	if !ok {
		t.Fatalf("expected monitor snapshot, got %+v", updated.Resources)
	}
	l4, ok := monitor["l4"].(map[string]any)
	if !ok || l4["agent_dispatchable"] != true {
		t.Fatalf("expected l4 agent snapshot, got %+v", monitor)
	}
	claude, ok := l4["claude_code"].(map[string]any)
	if !ok || claude["auth_status"] != "logged_in" || claude["ready"] != true {
		t.Fatalf("expected claude l4 snapshot, got %+v", l4)
	}
	l5, ok := monitor["l5"].(map[string]any)
	if !ok {
		t.Fatalf("expected l5 audit snapshot, got %+v", monitor)
	}
	gitAudit, ok := l5["git"].(map[string]any)
	if !ok || gitAudit["installed"] != true {
		t.Fatalf("expected git audit in l5 snapshot, got %+v", l5)
	}
	if updated.Resources["agent_environment_checked_at"] != checkedAt.Format(time.RFC3339) {
		t.Fatalf("expected updated agent environment timestamp, got %+v", updated.Resources)
	}
}

func TestTestMachineConnectionPreservesDetectedPlatformForWebsocketRuntimeMachines(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(transportinfra.NewWebsocketListenerHandler(transportinfra.ListenerHandlerOptions{}))
	defer server.Close()

	machineID := uuid.New()
	orgID := uuid.New()
	repo := &stubRepository{
		machine: domain.Machine{
			ID:                 machineID,
			OrganizationID:     orgID,
			Name:               "listener-01",
			Host:               "listener.internal",
			Port:               443,
			Status:             domain.MachineStatusOnline,
			ConnectionMode:     domain.MachineConnectionModeWSListener,
			AdvertisedEndpoint: stringPointer(websocketURL(server.URL)),
			DetectedOS:         domain.MachineDetectedOSLinux,
			DetectedArch:       domain.MachineDetectedArchAMD64,
			DetectionStatus:    domain.MachineDetectionStatusOK,
			Resources: map[string]any{
				"transport":         domain.MachineConnectionModeWSListener.String(),
				"detected_os":       domain.MachineDetectedOSLinux.String(),
				"detected_arch":     domain.MachineDetectedArchAMD64.String(),
				"detection_status":  domain.MachineDetectionStatusOK.String(),
				"cpu_usage_percent": 61.2,
			},
		},
	}
	tester := transportinfra.NewTester(transportinfra.NewResolver(nil, nil))
	svc := New(repo, stubExecutableResolver{}, tester)

	updated, probe, err := svc.TestMachineConnection(context.Background(), machineID)
	if err != nil {
		t.Fatalf("TestMachineConnection() error = %v", err)
	}
	if probe.Transport != domain.MachineConnectionModeWSListener.String() {
		t.Fatalf("TestMachineConnection().probe.Transport = %q", probe.Transport)
	}
	if repo.recordedMachineProbe == nil {
		t.Fatal("expected machine probe to be recorded")
	}
	if repo.recordedMachineProbe.DetectedOS != domain.MachineDetectedOSLinux ||
		repo.recordedMachineProbe.DetectedArch != domain.MachineDetectedArchAMD64 ||
		repo.recordedMachineProbe.DetectionStatus != domain.MachineDetectionStatusOK {
		t.Fatalf("recorded probe detection metadata = %+v", repo.recordedMachineProbe)
	}
	if updated.DetectedOS != domain.MachineDetectedOSLinux ||
		updated.DetectedArch != domain.MachineDetectedArchAMD64 ||
		updated.DetectionStatus != domain.MachineDetectionStatusOK {
		t.Fatalf("updated machine detection metadata = %+v", updated)
	}
}

func TestRequestAgentPausePersistsPauseRequestedState(t *testing.T) {
	agentID := uuid.New()
	runID := uuid.New()
	ticketID := uuid.New()
	repo := &stubRepository{
		agent: domain.Agent{
			ID:                  agentID,
			Name:                "worker-1",
			RuntimeControlState: domain.AgentRuntimeControlStateActive,
			Runtime: &domain.AgentRuntime{
				CurrentRunID:    &runID,
				Status:          domain.AgentStatusRunning,
				CurrentTicketID: &ticketID,
				RuntimePhase:    domain.AgentRuntimePhaseReady,
			},
		},
	}
	svc := New(repo, stubExecutableResolver{}, nil)

	item, err := svc.RequestAgentPause(context.Background(), agentID)
	if err != nil {
		t.Fatalf("RequestAgentPause returned error: %v", err)
	}
	if item.RuntimeControlState != domain.AgentRuntimeControlStatePauseRequested {
		t.Fatalf("expected pause_requested state, got %+v", item)
	}
	if repo.updatedRuntimeControl == nil || repo.updatedRuntimeControl.RuntimeControlState != domain.AgentRuntimeControlStatePauseRequested {
		t.Fatalf("expected repo runtime control update, got %+v", repo.updatedRuntimeControl)
	}
}

func TestRequestAgentInterruptPersistsInterruptRequestedState(t *testing.T) {
	agentID := uuid.New()
	runID := uuid.New()
	ticketID := uuid.New()
	repo := &stubRepository{
		agent: domain.Agent{
			ID:                  agentID,
			Name:                "worker-1",
			RuntimeControlState: domain.AgentRuntimeControlStateActive,
			Runtime: &domain.AgentRuntime{
				CurrentRunID:    &runID,
				Status:          domain.AgentStatusRunning,
				CurrentTicketID: &ticketID,
				RuntimePhase:    domain.AgentRuntimePhaseExecuting,
			},
		},
	}
	svc := New(repo, stubExecutableResolver{}, nil)

	item, err := svc.RequestAgentInterrupt(context.Background(), agentID)
	if err != nil {
		t.Fatalf("RequestAgentInterrupt returned error: %v", err)
	}
	if item.RuntimeControlState != domain.AgentRuntimeControlStateInterruptRequested {
		t.Fatalf("expected interrupt_requested state, got %+v", item)
	}
	if repo.updatedRuntimeControl == nil || repo.updatedRuntimeControl.RuntimeControlState != domain.AgentRuntimeControlStateInterruptRequested {
		t.Fatalf("expected repo runtime control update, got %+v", repo.updatedRuntimeControl)
	}
}

func TestRequestAgentResumeRejectsPauseRequestedState(t *testing.T) {
	agentID := uuid.New()
	runID := uuid.New()
	ticketID := uuid.New()
	repo := &stubRepository{
		agent: domain.Agent{
			ID:                  agentID,
			Name:                "worker-1",
			RuntimeControlState: domain.AgentRuntimeControlStatePauseRequested,
			Runtime: &domain.AgentRuntime{
				CurrentRunID:    &runID,
				Status:          domain.AgentStatusClaimed,
				CurrentTicketID: &ticketID,
				RuntimePhase:    domain.AgentRuntimePhaseNone,
			},
		},
	}
	svc := New(repo, stubExecutableResolver{}, nil)

	_, err := svc.RequestAgentResume(context.Background(), agentID)
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected runtime control conflict, got %v", err)
	}
}

func TestRequestAgentInterruptRejectsPauseRequestedState(t *testing.T) {
	agentID := uuid.New()
	runID := uuid.New()
	ticketID := uuid.New()
	repo := &stubRepository{
		agent: domain.Agent{
			ID:                  agentID,
			Name:                "worker-1",
			RuntimeControlState: domain.AgentRuntimeControlStatePauseRequested,
			Runtime: &domain.AgentRuntime{
				CurrentRunID:    &runID,
				Status:          domain.AgentStatusClaimed,
				CurrentTicketID: &ticketID,
				RuntimePhase:    domain.AgentRuntimePhaseNone,
			},
		},
	}
	svc := New(repo, stubExecutableResolver{}, nil)

	_, err := svc.RequestAgentInterrupt(context.Background(), agentID)
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected runtime control conflict, got %v", err)
	}
}

func TestRequestAgentResumePersistsActiveState(t *testing.T) {
	agentID := uuid.New()
	runID := uuid.New()
	ticketID := uuid.New()
	repo := &stubRepository{
		agent: domain.Agent{
			ID:                  agentID,
			Name:                "worker-1",
			RuntimeControlState: domain.AgentRuntimeControlStatePaused,
			Runtime: &domain.AgentRuntime{
				CurrentRunID:    &runID,
				Status:          domain.AgentStatusRunning,
				CurrentTicketID: &ticketID,
				RuntimePhase:    domain.AgentRuntimePhaseReady,
			},
		},
	}
	svc := New(repo, stubExecutableResolver{}, nil)

	item, err := svc.RequestAgentResume(context.Background(), agentID)
	if err != nil {
		t.Fatalf("RequestAgentResume returned error: %v", err)
	}
	if item.RuntimeControlState != domain.AgentRuntimeControlStateActive {
		t.Fatalf("expected active state, got %+v", item)
	}
	if repo.updatedRuntimeControl == nil || repo.updatedRuntimeControl.RuntimeControlState != domain.AgentRuntimeControlStateActive {
		t.Fatalf("expected repo runtime control update, got %+v", repo.updatedRuntimeControl)
	}
}

func TestRetireAgentPersistsRetiredState(t *testing.T) {
	agentID := uuid.New()
	repo := &stubRepository{
		agent: domain.Agent{
			ID:                  agentID,
			Name:                "worker-1",
			RuntimeControlState: domain.AgentRuntimeControlStateActive,
		},
	}
	svc := New(repo, stubExecutableResolver{}, nil)

	item, err := svc.RetireAgent(context.Background(), agentID)
	if err != nil {
		t.Fatalf("RetireAgent returned error: %v", err)
	}
	if item.RuntimeControlState != domain.AgentRuntimeControlStateRetired {
		t.Fatalf("expected retired state, got %+v", item)
	}
	if repo.updatedRuntimeControl == nil || repo.updatedRuntimeControl.RuntimeControlState != domain.AgentRuntimeControlStateRetired {
		t.Fatalf("expected repo runtime control update, got %+v", repo.updatedRuntimeControl)
	}
}

func TestRetireAgentRejectsActiveRun(t *testing.T) {
	agentID := uuid.New()
	runID := uuid.New()
	repo := &stubRepository{
		agent: domain.Agent{
			ID:                  agentID,
			Name:                "worker-1",
			RuntimeControlState: domain.AgentRuntimeControlStateActive,
			Runtime: &domain.AgentRuntime{
				CurrentRunID: &runID,
				Status:       domain.AgentStatusRunning,
			},
		},
	}
	svc := New(repo, stubExecutableResolver{}, nil)

	_, err := svc.RetireAgent(context.Background(), agentID)
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected runtime control conflict, got %v", err)
	}
}

func TestAgentProviderAvailabilityHelpers(t *testing.T) {
	codexPath := "/usr/local/bin/codex"
	remoteCodexID := uuid.New()
	localCodexID := uuid.New()
	claudeID := uuid.New()
	checkedAt := time.Now().UTC()
	workspaceRoot := "/tmp/openase-remote"

	items := annotateAgentProvidersAvailability([]domain.AgentProvider{
		{
			ID:                  localCodexID,
			Name:                "OpenAI Codex",
			AdapterType:         domain.AgentProviderAdapterTypeCodexAppServer,
			CliCommand:          "codex",
			MachineHost:         domain.LocalMachineHost,
			MachineStatus:       domain.MachineStatusOnline,
			MachineResources:    providerAvailabilityResources(checkedAt, "codex", true, domain.MachineAgentAuthStatusLoggedIn, domain.MachineAgentAuthModeLogin, true),
			MachineAgentCLIPath: &codexPath,
		},
		{
			ID:                   remoteCodexID,
			Name:                 "Remote Codex",
			AdapterType:          domain.AgentProviderAdapterTypeCodexAppServer,
			CliCommand:           "codex",
			MachineID:            uuid.New(),
			MachineHost:          "10.0.0.25",
			MachineStatus:        domain.MachineStatusOnline,
			MachineWorkspaceRoot: &workspaceRoot,
			MachineResources: map[string]any{
				"monitor": map[string]any{
					"l4": map[string]any{
						"checked_at": checkedAt.Format(time.RFC3339),
						"codex": map[string]any{
							"installed":   true,
							"auth_status": string(domain.MachineAgentAuthStatusLoggedIn),
							"auth_mode":   string(domain.MachineAgentAuthModeLogin),
							"ready":       true,
						},
					},
				},
			},
		},
		{
			ID:            claudeID,
			Name:          "Claude Code",
			AdapterType:   domain.AgentProviderAdapterTypeClaudeCodeCLI,
			CliCommand:    "claude",
			MachineID:     uuid.New(),
			MachineHost:   "10.0.0.50",
			MachineStatus: domain.MachineStatusMaintenance,
		},
	})

	if !items[0].Available {
		t.Fatalf("expected local codex provider to be available, got %+v", items[0])
	}
	if !items[1].Available {
		t.Fatalf("expected remote codex provider to be available, got %+v", items[1])
	}
	if items[2].Available {
		t.Fatalf("expected maintenance provider to be unavailable, got %+v", items[2])
	}

	state, checked, reason := domain.ResolveAgentProviderAvailability(items[1], checkedAt)
	if state != domain.AgentProviderAvailabilityStateAvailable || checked == nil || reason != nil {
		t.Fatalf("ResolveAgentProviderAvailability(remote) = %q, %v, %v", state, checked, reason)
	}
	state, checked, reason = domain.ResolveAgentProviderAvailability(domain.AgentProvider{
		AdapterType:   domain.AgentProviderAdapterTypeClaudeCodeCLI,
		MachineStatus: domain.MachineStatusOnline,
	}, checkedAt)
	if state != domain.AgentProviderAvailabilityStateUnknown || checked != nil || reason == nil || *reason != "l4_snapshot_missing" {
		t.Fatalf("ResolveAgentProviderAvailability(missing snapshot) = %q, %v, %v", state, checked, reason)
	}
	if got := preferredAvailableProviderID(items); got == nil || *got != localCodexID {
		t.Fatalf("preferredAvailableProviderID() = %v, want %s", got, localCodexID)
	}
}

type stubExecutableResolver struct {
	paths map[string]string
}

func (r stubExecutableResolver) LookPath(name string) (string, error) {
	if value, ok := r.paths[name]; ok {
		return value, nil
	}

	return "", errors.New("not found")
}

type stubMachineTester struct {
	probe domain.MachineProbe
	err   error
}

func (s stubMachineTester) TestConnection(context.Context, domain.Machine) (domain.MachineProbe, error) {
	return s.probe, s.err
}

type stubMachineHealthCollector struct {
	reachability        domain.MachineReachability
	reachabilityErr     error
	systemResources     domain.MachineSystemResources
	systemResourcesErr  error
	gpuResources        domain.MachineGPUResources
	gpuResourcesErr     error
	agentEnvironment    domain.MachineAgentEnvironment
	agentEnvironmentErr error
	fullAudit           domain.MachineFullAudit
	fullAuditErr        error
}

func (s stubMachineHealthCollector) CollectReachability(context.Context, domain.Machine) (domain.MachineReachability, error) {
	return s.reachability, s.reachabilityErr
}

func (s stubMachineHealthCollector) CollectSystemResources(context.Context, domain.Machine) (domain.MachineSystemResources, error) {
	return s.systemResources, s.systemResourcesErr
}

func (s stubMachineHealthCollector) CollectGPUResources(context.Context, domain.Machine) (domain.MachineGPUResources, error) {
	return s.gpuResources, s.gpuResourcesErr
}

func (s stubMachineHealthCollector) CollectAgentEnvironment(context.Context, domain.Machine) (domain.MachineAgentEnvironment, error) {
	return s.agentEnvironment, s.agentEnvironmentErr
}

func (s stubMachineHealthCollector) CollectFullAudit(context.Context, domain.Machine) (domain.MachineFullAudit, error) {
	return s.fullAudit, s.fullAuditErr
}

type stubRepository struct {
	createdProvider        *domain.CreateAgentProvider
	updatedProvider        *domain.UpdateAgentProvider
	updatedAgent           *domain.UpdateAgent
	updatedRuntimeControl  *domain.UpdateAgentRuntimeControlState
	updatedOrganization    *domain.UpdateOrganization
	archivedOrganizationID uuid.UUID
	createdOrganization    domain.Organization
	createdProject         domain.Project
	organizations          []domain.Organization
	projects               []domain.Project
	projectRepos           []domain.ProjectRepo
	agents                 []domain.Agent
	agentRuns              []domain.AgentRun
	listedProviders        []domain.AgentProvider
	traceEntries           []domain.AgentTraceEntry
	stepEntries            []domain.AgentStepEntry
	provider               domain.AgentProvider
	agent                  domain.Agent
	machine                domain.Machine
	recordedMachineProbe   *domain.RecordMachineProbe
}

func (r *stubRepository) ListOrganizations(context.Context) ([]domain.Organization, error) {
	return append([]domain.Organization(nil), r.organizations...), nil
}

func (r *stubRepository) CreateOrganization(context.Context, domain.CreateOrganization) (domain.Organization, error) {
	return r.createdOrganization, nil
}

func (r *stubRepository) GetOrganization(context.Context, uuid.UUID) (domain.Organization, error) {
	return domain.Organization{}, nil
}

func (r *stubRepository) UpdateOrganization(_ context.Context, input domain.UpdateOrganization) (domain.Organization, error) {
	r.updatedOrganization = &input
	r.createdOrganization = domain.Organization{
		ID:                     input.ID,
		Name:                   input.Name,
		Slug:                   input.Slug,
		Status:                 r.createdOrganization.Status,
		DefaultAgentProviderID: input.DefaultAgentProviderID,
	}
	return r.createdOrganization, nil
}

func (r *stubRepository) ArchiveOrganization(_ context.Context, id uuid.UUID) (domain.Organization, error) {
	r.archivedOrganizationID = id
	return r.createdOrganization, nil
}

func (r *stubRepository) ListProjects(context.Context, uuid.UUID) ([]domain.Project, error) {
	return append([]domain.Project(nil), r.projects...), nil
}

func (r *stubRepository) ListMachines(context.Context, uuid.UUID) ([]domain.Machine, error) {
	return nil, nil
}

func (r *stubRepository) CreateMachine(context.Context, domain.CreateMachine) (domain.Machine, error) {
	return domain.Machine{}, nil
}

func (r *stubRepository) GetMachine(context.Context, uuid.UUID) (domain.Machine, error) {
	return r.machine, nil
}

func (r *stubRepository) UpdateMachine(context.Context, domain.UpdateMachine) (domain.Machine, error) {
	return domain.Machine{}, nil
}

func (r *stubRepository) DeleteMachine(context.Context, uuid.UUID) (domain.Machine, error) {
	return domain.Machine{}, nil
}

func (r *stubRepository) RecordMachineProbe(_ context.Context, input domain.RecordMachineProbe) error {
	return r.recordMachineProbe(input)
}

func (r *stubRepository) recordMachineProbe(input domain.RecordMachineProbe) error {
	detectedOS, err := domain.ParseStoredMachineDetectedOS(input.DetectedOS.String())
	if err != nil {
		return err
	}
	detectedArch, err := domain.ParseStoredMachineDetectedArch(input.DetectedArch.String())
	if err != nil {
		return err
	}
	detectionStatus, err := domain.ParseStoredMachineDetectionStatus(input.DetectionStatus.String())
	if err != nil {
		return err
	}
	input.DetectedOS = detectedOS
	input.DetectedArch = detectedArch
	input.DetectionStatus = detectionStatus
	r.recordedMachineProbe = &input
	r.machine.Status = input.Status
	r.machine.LastHeartbeatAt = &input.LastHeartbeatAt
	r.machine.DetectedOS = input.DetectedOS
	r.machine.DetectedArch = input.DetectedArch
	r.machine.DetectionStatus = input.DetectionStatus
	r.machine.Resources = cloneTestResources(input.Resources)
	return nil
}

func websocketURL(raw string) string {
	switch {
	case strings.HasPrefix(raw, "https://"):
		return "wss://" + strings.TrimPrefix(raw, "https://")
	case strings.HasPrefix(raw, "http://"):
		return "ws://" + strings.TrimPrefix(raw, "http://")
	default:
		return raw
	}
}

func stringPointer(value string) *string {
	return &value
}

func (r *stubRepository) CreateProject(context.Context, domain.CreateProject) (domain.Project, error) {
	return r.createdProject, nil
}

func (r *stubRepository) GetProject(context.Context, uuid.UUID) (domain.Project, error) {
	if len(r.projects) > 0 {
		return r.projects[0], nil
	}
	return r.createdProject, nil
}

func (r *stubRepository) UpdateProject(context.Context, domain.UpdateProject) (domain.Project, error) {
	return domain.Project{}, nil
}

func (r *stubRepository) ArchiveProject(context.Context, uuid.UUID) (domain.Project, error) {
	return domain.Project{}, nil
}

func (r *stubRepository) ListAgentProviders(context.Context, uuid.UUID) ([]domain.AgentProvider, error) {
	return append([]domain.AgentProvider(nil), r.listedProviders...), nil
}

func (r *stubRepository) CreateAgentProvider(_ context.Context, input domain.CreateAgentProvider) (domain.AgentProvider, error) {
	r.createdProvider = &input

	return domain.AgentProvider{
		ID:             uuid.New(),
		OrganizationID: input.OrganizationID,
		MachineID:      input.MachineID,
		Name:           input.Name,
		AdapterType:    input.AdapterType,
		CliCommand:     input.CliCommand,
		CliArgs:        append([]string(nil), input.CliArgs...),
		ModelName:      input.ModelName,
		AuthConfig:     input.AuthConfig,
	}, nil
}

func (r *stubRepository) GetAgentProvider(context.Context, uuid.UUID) (domain.AgentProvider, error) {
	return r.provider, nil
}

func (r *stubRepository) UpdateAgentProvider(_ context.Context, input domain.UpdateAgentProvider) (domain.AgentProvider, error) {
	r.updatedProvider = &input

	return domain.AgentProvider{
		ID:             input.ID,
		OrganizationID: input.OrganizationID,
		MachineID:      input.MachineID,
		Name:           input.Name,
		AdapterType:    input.AdapterType,
		CliCommand:     input.CliCommand,
		CliArgs:        append([]string(nil), input.CliArgs...),
		ModelName:      input.ModelName,
		AuthConfig:     input.AuthConfig,
	}, nil
}

func (r *stubRepository) ListAgents(context.Context, uuid.UUID) ([]domain.Agent, error) {
	return append([]domain.Agent(nil), r.agents...), nil
}

func (r *stubRepository) ListAgentRuns(context.Context, uuid.UUID) ([]domain.AgentRun, error) {
	return append([]domain.AgentRun(nil), r.agentRuns...), nil
}

func (r *stubRepository) ListActivityEvents(context.Context, domain.ListActivityEvents) ([]domain.ActivityEvent, error) {
	return nil, nil
}

func (r *stubRepository) ListAgentOutput(context.Context, domain.ListAgentOutput) ([]domain.AgentOutputEntry, error) {
	return nil, nil
}

func (r *stubRepository) ListAgentSteps(context.Context, domain.ListAgentSteps) ([]domain.AgentStepEntry, error) {
	return nil, nil
}

func (r *stubRepository) ListAgentRunTraceEntries(context.Context, domain.ListAgentRunTraceEntries) ([]domain.AgentTraceEntry, error) {
	return append([]domain.AgentTraceEntry(nil), r.traceEntries...), nil
}

func (r *stubRepository) ListAgentRunStepEntries(context.Context, domain.ListAgentRunStepEntries) ([]domain.AgentStepEntry, error) {
	return append([]domain.AgentStepEntry(nil), r.stepEntries...), nil
}

func (r *stubRepository) CreateAgent(context.Context, domain.CreateAgent) (domain.Agent, error) {
	return domain.Agent{}, nil
}

func (r *stubRepository) GetAgent(context.Context, uuid.UUID) (domain.Agent, error) {
	return r.agent, nil
}

func (r *stubRepository) GetAgentRun(context.Context, uuid.UUID) (domain.AgentRun, error) {
	return domain.AgentRun{}, nil
}

func (r *stubRepository) UpdateAgent(_ context.Context, input domain.UpdateAgent) (domain.Agent, error) {
	r.updatedAgent = &input
	r.agent.ProviderID = input.ProviderID
	r.agent.Name = input.Name
	return r.agent, nil
}

func (r *stubRepository) UpdateAgentRuntimeControlState(_ context.Context, input domain.UpdateAgentRuntimeControlState) (domain.Agent, error) {
	r.updatedRuntimeControl = &input
	r.agent.RuntimeControlState = input.RuntimeControlState
	return r.agent, nil
}

func (r *stubRepository) DeleteAgent(context.Context, uuid.UUID) (domain.Agent, error) {
	return domain.Agent{}, nil
}

func (r *stubRepository) ListProjectRepos(context.Context, uuid.UUID) ([]domain.ProjectRepo, error) {
	return append([]domain.ProjectRepo(nil), r.projectRepos...), nil
}

func (r *stubRepository) CreateProjectRepo(context.Context, domain.CreateProjectRepo) (domain.ProjectRepo, error) {
	return domain.ProjectRepo{}, nil
}

func (r *stubRepository) GetProjectRepo(context.Context, uuid.UUID, uuid.UUID) (domain.ProjectRepo, error) {
	return domain.ProjectRepo{}, nil
}

func (r *stubRepository) UpdateProjectRepo(context.Context, domain.UpdateProjectRepo) (domain.ProjectRepo, error) {
	return domain.ProjectRepo{}, nil
}

func (r *stubRepository) DeleteProjectRepo(context.Context, uuid.UUID, uuid.UUID) (domain.ProjectRepo, error) {
	return domain.ProjectRepo{}, nil
}

func (r *stubRepository) ListTicketRepoScopes(context.Context, uuid.UUID, uuid.UUID) ([]domain.TicketRepoScope, error) {
	return nil, nil
}

func (r *stubRepository) CreateTicketRepoScope(context.Context, domain.CreateTicketRepoScope) (domain.TicketRepoScope, error) {
	return domain.TicketRepoScope{}, nil
}

func (r *stubRepository) GetTicketRepoScope(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (domain.TicketRepoScope, error) {
	return domain.TicketRepoScope{}, nil
}

func (r *stubRepository) UpdateTicketRepoScope(context.Context, domain.UpdateTicketRepoScope) (domain.TicketRepoScope, error) {
	return domain.TicketRepoScope{}, nil
}

func (r *stubRepository) DeleteTicketRepoScope(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (domain.TicketRepoScope, error) {
	return domain.TicketRepoScope{}, nil
}

func (r *stubRepository) GetWorkspaceDashboardSummary(context.Context) (domain.WorkspaceDashboardSummary, error) {
	return domain.WorkspaceDashboardSummary{}, nil
}

func (r *stubRepository) GetOrganizationDashboardSummary(context.Context, uuid.UUID) (domain.OrganizationDashboardSummary, error) {
	return domain.OrganizationDashboardSummary{}, nil
}

func (r *stubRepository) GetOrganizationTokenUsage(context.Context, domain.GetOrganizationTokenUsage) (domain.OrganizationTokenUsageReport, error) {
	return domain.OrganizationTokenUsageReport{}, nil
}

func (r *stubRepository) GetProjectTokenUsage(context.Context, domain.GetProjectTokenUsage) (domain.ProjectTokenUsageReport, error) {
	return domain.ProjectTokenUsageReport{}, nil
}

var _ Repository = (*stubRepository)(nil)

func equalStrings(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}

	return true
}

func cloneTestResources(raw map[string]any) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}

	cloned := make(map[string]any, len(raw))
	for key, value := range raw {
		cloned[key] = value
	}

	return cloned
}

func providerAvailabilityResources(
	checkedAt time.Time,
	entryName string,
	installed bool,
	authStatus domain.MachineAgentAuthStatus,
	authMode domain.MachineAgentAuthMode,
	ready bool,
) map[string]any {
	return map[string]any{
		"monitor": map[string]any{
			"l4": map[string]any{
				"checked_at": checkedAt.UTC().Format(time.RFC3339),
				entryName: map[string]any{
					"installed":   installed,
					"auth_status": string(authStatus),
					"auth_mode":   string(authMode),
					"ready":       ready,
				},
			},
		},
	}
}

type stubProjectStatusBootstrapper struct {
	projectID uuid.UUID
}

func (s *stubProjectStatusBootstrapper) BootstrapProjectStatuses(_ context.Context, projectID uuid.UUID) error {
	s.projectID = projectID
	return nil
}
