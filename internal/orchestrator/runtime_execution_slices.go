package orchestrator

type runtimeAssignmentSelectionSlice struct {
	launcher *RuntimeLauncher
}

type runtimeWorkspacePreparationSlice struct {
	launcher *RuntimeLauncher
}

type runtimeProcessLifecycleSlice struct {
	launcher *RuntimeLauncher
}

type runtimeExecutionSlice struct {
	launcher *RuntimeLauncher
}

type runtimeRecoverySlice struct {
	launcher *RuntimeLauncher
}

type runtimeEventPersistenceSlice struct {
	launcher *RuntimeLauncher
}

func (l *RuntimeLauncher) selectionSlice() runtimeAssignmentSelectionSlice {
	return runtimeAssignmentSelectionSlice{launcher: l}
}

func (l *RuntimeLauncher) workspaceSlice() runtimeWorkspacePreparationSlice {
	return runtimeWorkspacePreparationSlice{launcher: l}
}

func (l *RuntimeLauncher) processSlice() runtimeProcessLifecycleSlice {
	return runtimeProcessLifecycleSlice{launcher: l}
}

func (l *RuntimeLauncher) executionSlice() runtimeExecutionSlice {
	return runtimeExecutionSlice{launcher: l}
}

func (l *RuntimeLauncher) recoverySlice() runtimeRecoverySlice {
	return runtimeRecoverySlice{launcher: l}
}

func (l *RuntimeLauncher) eventSlice() runtimeEventPersistenceSlice {
	return runtimeEventPersistenceSlice{launcher: l}
}
