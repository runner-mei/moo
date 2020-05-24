package authz

import (
	"context"
	"errors"
)

// define a set of errors
var (
	ErrFieldIncomplete = errors.New("incomplete fields")
	ErrEmptyStructure  = errors.New("empty structure")
)

// Permissions is the set of Permission
type Permissions []Permission

// Permission is used to define permission control information
type Permission struct {
	// AuthorizedRoles defines roles that allow access to specified resource
	// Accepted type: non-empty string, *
	//      *: means any role, but visitors should have at least one role,
	//      non-empty string: specified role
	AuthorizedRoles []int64 `json:"authorized_roles" yaml:"authorized_roles"`

	// ForbiddenRoles defines roles that not allow access to specified resource
	// ForbiddenRoles has a higher priority than AuthorizedRoles
	// Accepted type: non-empty string, *
	//      *: means any role, but visitors should have at least one role,
	//      non-empty string: specified role
	//
	ForbiddenRoles []int64 `json:"forbidden_roles" yaml:"forbidden_roles"`

	// AllowAnyone has a higher priority than ForbiddenRoles/AuthorizedRoles
	// If set to true, anyone will be able to pass authentication.
	// Note that this will include people without any role.
	AllowAnyone bool `json:"allow_anyone" yaml:"allow_anyone"`
}

// IsGranted is used to determine whether the given role can pass the authentication of *Permission.
func (p *Permission) IsGranted(roles []int64) (PermissionState, error) {
	if p.AllowAnyone {
		return PermissionGranted, nil
	}

	if len(roles) == 0 {
		return PermissionUngranted, nil
	}

	for _, role := range roles {
		for _, forbidden := range p.ForbiddenRoles {
			if role == forbidden {
				return PermissionUngranted, nil
			}
		}
		for _, authorized := range p.AuthorizedRoles {
			if role == authorized {
				return PermissionGranted, nil
			}
		}
	}
	return PermissionUngranted, nil
}

type Resource = string

type Authorizer interface {
	IsGranted(ctx context.Context, res Resource, roles []int64) (PermissionState, error)
}

type EmptyAuthorizer struct{}

func (EmptyAuthorizer) IsGranted(ctx context.Context, res Resource, roles []int64) (PermissionState, error) {
	return PermissionGranted, nil
}
