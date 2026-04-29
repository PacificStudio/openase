package catalog

func IsLocalMachineIdentity(name, host string, mode MachineConnectionMode) bool {
	return name == LocalMachineName && host == LocalMachineHost && mode == MachineConnectionModeLocal
}

// DefaultMachineStatus returns the stored machine status to use when the
// caller does not provide one explicitly. Remote machines begin offline until
// connectivity or health checks prove they are schedulable; maintenance is
// reserved for an operator-controlled gate.
func DefaultMachineStatus(isLocal bool) MachineStatus {
	if isLocal {
		return MachineStatusOnline
	}
	return MachineStatusOffline
}

func MachineInManualMaintenance(status MachineStatus) bool {
	return status == MachineStatusMaintenance
}

func ApplyMachineMaintenanceGate(currentStatus, inferredStatus MachineStatus) MachineStatus {
	if MachineInManualMaintenance(currentStatus) {
		return MachineStatusMaintenance
	}
	return inferredStatus
}

func InferMachineConnectionSuccessStatus(currentStatus MachineStatus) MachineStatus {
	return ApplyMachineMaintenanceGate(currentStatus, MachineStatusOnline)
}

func InferMachineConnectionFailureStatus(current Machine) MachineStatus {
	inferred := MachineStatusOffline
	if current.Host == LocalMachineHost {
		inferred = MachineStatusDegraded
	}
	return ApplyMachineMaintenanceGate(current.Status, inferred)
}

func InferMachineRefreshedHealthStatus(currentStatus MachineStatus, inferredStatus MachineStatus) MachineStatus {
	return ApplyMachineMaintenanceGate(currentStatus, inferredStatus)
}
