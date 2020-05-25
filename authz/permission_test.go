package authz

import (
	"reflect"
	"testing"
)

func TestPermission_IsValid(t *testing.T) {
	tests := []struct {
		name    string
		p       *Permission
		wantErr bool
	}{
		{
			name: "test0",
			p: &Permission{
				AuthorizedRoles: []int64{},
				ForbiddenRoles:  []int64{},
				AllowAnyone:     false,
			},
			wantErr: true,
		},
		{
			name: "test1",
			p: &Permission{
				AuthorizedRoles: []int64{},
				ForbiddenRoles:  []int64{},
				AllowAnyone:     true,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		if err := tt.p.IsValid(); (err != nil) != tt.wantErr {
			t.Errorf("%q. Permission.IsValid() error = %v, wantErr %v", tt.name, err, tt.wantErr)
		}
	}
}

func TestPermission_IsGranted(t *testing.T) {
	type args struct {
		roles []string
	}
	tests := []struct {
		name    string
		p       *Permission
		args    args
		want    PermissionState
		wantErr bool
	}{
		{
			name: "test0",
			p: &Permission{
				AllowAnyone: true,
			},
			args:    args{},
			want:    PermissionGranted,
			wantErr: false,
		},
		{
			name: "test1",
			p: &Permission{
				AuthorizedRoles: []int64{"editor"},
				AllowAnyone:     false,
			},
			args:    args{roles: []int64{"editor"}},
			want:    PermissionGranted,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		got, err := tt.p.IsGranted(tt.args.roles)
		if (err != nil) != tt.wantErr {
			t.Errorf("%q. Permission.IsGranted() error = %v, wantErr %v", tt.name, err, tt.wantErr)
			continue
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%q. Permission.IsGranted() = %v, want %v", tt.name, got, tt.want)
		}
	}
}
