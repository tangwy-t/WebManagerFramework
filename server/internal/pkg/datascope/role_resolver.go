package datascope

import (
	"context"
)

// RoleDimensionResolver resolves the "role" dimension data scope for menu filtering.
type RoleDimensionResolver struct {
	roleMenuRepo RoleMenuRepoInterface
}

// NewRoleDimensionResolver creates a RoleDimensionResolver.
func NewRoleDimensionResolver(roleMenuRepo RoleMenuRepoInterface) *RoleDimensionResolver {
	return &RoleDimensionResolver{roleMenuRepo: roleMenuRepo}
}

// DimensionType returns "role".
func (r *RoleDimensionResolver) DimensionType() string { return DimRole }

// Resolve resolves the accessible menu ID set based on the scope level.
// ScopeAll → nil (all allowed); ScopeCustom → role menu IDs from DB; other levels → nil (all).
func (r *RoleDimensionResolver) Resolve(ctx context.Context, level int8, selfID uint64, userID uint64) (*ResolvedDimension, error) {
	dim := &ResolvedDimension{Level: level, SelfID: selfID}
	switch level {
	case ScopeAll:
		dim.AllowedIDs = nil // nil = all
	case ScopeCustom:
		ids, err := r.roleMenuRepo.FindRoleMenuIDs(ctx, userID)
		if err != nil {
			return nil, err
		}
		dim.AllowedIDs = ids
	}
	// nil = all for other levels
	return dim, nil
}
