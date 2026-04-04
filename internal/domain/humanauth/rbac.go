package humanauth

import "sort"

type RoleDefinition struct {
	Key         RoleKey
	Permissions []PermissionKey
}

func BuiltinRoles() map[RoleKey]RoleDefinition {
	allProjectAdmin := []PermissionKey{
		PermissionOrgRead,
		PermissionProjectRead,
		PermissionProjectUpdate,
		PermissionProjectDelete,
		PermissionRepoRead,
		PermissionRepoManage,
		PermissionTicketRead,
		PermissionTicketCreate,
		PermissionTicketUpdate,
		PermissionTicketComment,
		PermissionWorkflowRead,
		PermissionWorkflowManage,
		PermissionSkillRead,
		PermissionSkillManage,
		PermissionAgentRead,
		PermissionAgentManage,
		PermissionJobRead,
		PermissionJobManage,
		PermissionSecurityRead,
		PermissionSecurityManage,
		PermissionProposalApprove,
		PermissionRBACManage,
	}
	return map[RoleKey]RoleDefinition{
		RoleInstanceAdmin: {
			Key: RoleInstanceAdmin,
			Permissions: append([]PermissionKey{
				PermissionOrgUpdate,
			}, allProjectAdmin...),
		},
		RoleOrgOwner: {
			Key: RoleOrgOwner,
			Permissions: append([]PermissionKey{
				PermissionOrgRead,
				PermissionOrgUpdate,
			}, allProjectAdmin...),
		},
		RoleOrgAdmin: {
			Key: RoleOrgAdmin,
			Permissions: append([]PermissionKey{
				PermissionOrgRead,
				PermissionOrgUpdate,
			}, allProjectAdmin...),
		},
		RoleOrgMember: {
			Key: RoleOrgMember,
			Permissions: []PermissionKey{
				PermissionOrgRead,
				PermissionProjectRead,
				PermissionRepoRead,
				PermissionTicketRead,
				PermissionTicketCreate,
				PermissionTicketUpdate,
				PermissionTicketComment,
				PermissionWorkflowRead,
				PermissionSkillRead,
				PermissionAgentRead,
				PermissionJobRead,
				PermissionSecurityRead,
			},
		},
		RoleProjectAdmin: {
			Key:         RoleProjectAdmin,
			Permissions: allProjectAdmin,
		},
		RoleProjectOperator: {
			Key: RoleProjectOperator,
			Permissions: []PermissionKey{
				PermissionOrgRead,
				PermissionProjectRead,
				PermissionProjectUpdate,
				PermissionRepoRead,
				PermissionRepoManage,
				PermissionTicketRead,
				PermissionTicketCreate,
				PermissionTicketUpdate,
				PermissionTicketComment,
				PermissionWorkflowRead,
				PermissionWorkflowManage,
				PermissionSkillRead,
				PermissionSkillManage,
				PermissionAgentRead,
				PermissionAgentManage,
				PermissionJobRead,
				PermissionJobManage,
				PermissionSecurityRead,
			},
		},
		RoleProjectReviewer: {
			Key: RoleProjectReviewer,
			Permissions: []PermissionKey{
				PermissionOrgRead,
				PermissionProjectRead,
				PermissionRepoRead,
				PermissionTicketRead,
				PermissionTicketComment,
				PermissionWorkflowRead,
				PermissionSkillRead,
				PermissionAgentRead,
				PermissionJobRead,
				PermissionProposalApprove,
			},
		},
		RoleProjectMember: {
			Key: RoleProjectMember,
			Permissions: []PermissionKey{
				PermissionOrgRead,
				PermissionProjectRead,
				PermissionProjectUpdate,
				PermissionRepoRead,
				PermissionTicketRead,
				PermissionTicketCreate,
				PermissionTicketUpdate,
				PermissionTicketComment,
				PermissionWorkflowRead,
				PermissionSkillRead,
				PermissionAgentRead,
				PermissionJobRead,
			},
		},
		RoleProjectViewer: {
			Key: RoleProjectViewer,
			Permissions: []PermissionKey{
				PermissionOrgRead,
				PermissionProjectRead,
				PermissionRepoRead,
				PermissionTicketRead,
				PermissionWorkflowRead,
				PermissionSkillRead,
				PermissionAgentRead,
				PermissionJobRead,
				PermissionSecurityRead,
			},
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
