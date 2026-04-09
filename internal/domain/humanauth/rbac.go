package humanauth

import "sort"

type RoleDefinition struct {
	Key         RoleKey
	Permissions []PermissionKey
}

func BuiltinRoles() map[RoleKey]RoleDefinition {
	projectViewer := []PermissionKey{
		PermissionOrgRead,
		PermissionProjectRead,
		PermissionRepoRead,
		PermissionTicketRead,
		PermissionTicketCommentRead,
		PermissionProjectUpdateRead,
		PermissionWorkflowRead,
		PermissionHarnessRead,
		PermissionStatusRead,
		PermissionSkillRead,
		PermissionAgentRead,
		PermissionJobRead,
		PermissionSecurityRead,
		PermissionNotificationRead,
		PermissionConversationRead,
	}
	projectMember := appendPermissionLists(
		projectViewer,
		[]PermissionKey{
			PermissionTicketCreate,
			PermissionTicketUpdate,
			PermissionTicketCommentCreate,
			PermissionTicketCommentUpdate,
			PermissionProjectUpdateCreate,
			PermissionProjectUpdateUpdate,
			PermissionConversationCreate,
			PermissionConversationUpdate,
		},
	)
	projectOperator := appendPermissionLists(
		projectMember,
		[]PermissionKey{
			PermissionProjectUpdate,
			PermissionRepoCreate,
			PermissionRepoUpdate,
			PermissionRepoDelete,
			PermissionWorkflowCreate,
			PermissionWorkflowUpdate,
			PermissionWorkflowDelete,
			PermissionHarnessUpdate,
			PermissionStatusCreate,
			PermissionStatusUpdate,
			PermissionStatusDelete,
			PermissionSkillCreate,
			PermissionSkillUpdate,
			PermissionSkillDelete,
			PermissionAgentCreate,
			PermissionAgentUpdate,
			PermissionAgentDelete,
			PermissionAgentControl,
			PermissionJobCreate,
			PermissionJobUpdate,
			PermissionJobDelete,
			PermissionJobTrigger,
			PermissionNotificationCreate,
			PermissionNotificationUpdate,
			PermissionNotificationDelete,
		},
	)
	projectAdmin := appendPermissionLists(
		projectOperator,
		[]PermissionKey{
			PermissionProjectDelete,
			PermissionSecurityUpdate,
			PermissionConversationDelete,
			PermissionProposalApprove,
			PermissionRBACManage,
		},
	)
	orgOperator := appendPermissionLists(
		projectAdmin,
		[]PermissionKey{
			PermissionProjectCreate,
			PermissionMachineRead,
			PermissionMachineCreate,
			PermissionMachineUpdate,
			PermissionMachineDelete,
			PermissionProviderRead,
			PermissionProviderCreate,
			PermissionProviderUpdate,
			PermissionProviderDelete,
		},
	)
	instanceAdmin := appendPermissionLists(
		orgOperator,
		[]PermissionKey{
			PermissionOrgCreate,
			PermissionOrgUpdate,
			PermissionOrgDelete,
		},
	)
	orgAdmin := appendPermissionLists(
		orgOperator,
		[]PermissionKey{
			PermissionOrgRead,
			PermissionOrgUpdate,
		},
	)
	orgOwner := appendPermissionLists(
		orgAdmin,
		[]PermissionKey{
			PermissionOrgDelete,
		},
	)

	return map[RoleKey]RoleDefinition{
		RoleInstanceAdmin: {
			Key:         RoleInstanceAdmin,
			Permissions: instanceAdmin,
		},
		RoleOrgOwner: {
			Key:         RoleOrgOwner,
			Permissions: orgOwner,
		},
		RoleOrgAdmin: {
			Key:         RoleOrgAdmin,
			Permissions: orgAdmin,
		},
		RoleOrgMember: {
			Key: RoleOrgMember,
			Permissions: appendPermissionLists(
				projectMember,
				[]PermissionKey{
					PermissionMachineRead,
					PermissionProviderRead,
				},
			),
		},
		RoleProjectAdmin: {
			Key:         RoleProjectAdmin,
			Permissions: projectAdmin,
		},
		RoleProjectOperator: {
			Key:         RoleProjectOperator,
			Permissions: projectOperator,
		},
		RoleProjectReviewer: {
			Key: RoleProjectReviewer,
			Permissions: appendPermissionLists(
				projectViewer,
				[]PermissionKey{
					PermissionTicketCommentCreate,
					PermissionTicketCommentUpdate,
					PermissionConversationCreate,
					PermissionProposalApprove,
				},
			),
		},
		RoleProjectMember: {
			Key:         RoleProjectMember,
			Permissions: projectMember,
		},
		RoleProjectViewer: {
			Key:         RoleProjectViewer,
			Permissions: projectViewer,
		},
	}
}

func PermissionsForRoles(roles []RoleKey) []PermissionKey {
	roleDefs := BuiltinRoles()
	seen := map[PermissionKey]struct{}{}
	for _, role := range roles {
		def, ok := roleDefs[role]
		if !ok {
			continue
		}
		for _, permission := range def.Permissions {
			seen[permission] = struct{}{}
		}
	}
	permissions := make([]PermissionKey, 0, len(seen))
	for permission := range seen {
		permissions = append(permissions, permission)
	}
	sort.Slice(permissions, func(i, j int) bool { return permissions[i] < permissions[j] })
	return permissions
}

func appendPermissionLists(base []PermissionKey, extra []PermissionKey) []PermissionKey {
	seen := map[PermissionKey]struct{}{}
	combined := make([]PermissionKey, 0, len(base)+len(extra))
	for _, permission := range append(append([]PermissionKey{}, base...), extra...) {
		if _, ok := seen[permission]; ok {
			continue
		}
		seen[permission] = struct{}{}
		combined = append(combined, permission)
	}
	sort.Slice(combined, func(i, j int) bool { return combined[i] < combined[j] })
	return combined
}
