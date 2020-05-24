package authz

import "testing"

func TestPermissionState_IsLooselyGranted(t *testing.T) {
    tests := []struct {
        name  string
        state PermissionState
        want  bool
    }{
        {
            name:  "test0",
            state: PermissionNeglected,
            want:  true,
        },
    }
    for _, tt := range tests {
        if got := tt.state.IsLooselyGranted(); got != tt.want {
            t.Errorf("%q. PermissionState.IsLooselyGranted() = %v, want %v", tt.name, got, tt.want)
        }
    }
}

func TestPermissionState_IsNeglected(t *testing.T) {
    tests := []struct {
        name  string
        state PermissionState
        want  bool
    }{
        {
            name:  "test0",
            state: PermissionUngranted,
            want:  false,
        },
        {
            name:  "test1",
            state: PermissionNeglected,
            want:  true,
        },
    }
    for _, tt := range tests {
        if got := tt.state.IsNeglected(); got != tt.want {
            t.Errorf("%q. PermissionState.IsNeglected() = %v, want %v", tt.name, got, tt.want)
        }
    }
}

func TestPermissionState_IsGranted(t *testing.T) {
    tests := []struct {
        name  string
        state PermissionState
        want  bool
    }{
        {
            name:  "test0",
            state: PermissionUngranted,
            want:  false,
        },
        {
            name:  "test1",
            state: PermissionUnknown,
            want:  false,
        },
        {
            name:  "test2",
            state: PermissionNeglected,
            want:  false,
        },
    }
    for _, tt := range tests {
        if got := tt.state.IsGranted(); got != tt.want {
            t.Errorf("%q. PermissionState.IsGranted() = %v, want %v", tt.name, got, tt.want)
        }
    }
}

func TestPermissionState_String(t *testing.T) {
    tests := []struct {
        name  string
        state PermissionState
        want  string
    }{
        {
            name:  "test0",
            state: PermissionNeglected,
            want:  "Permission Neglected",
        },
    }
    for _, tt := range tests {
        if got := tt.state.String(); got != tt.want {
            t.Errorf("%q. PermissionState.String() = %v, want %v", tt.name, got, tt.want)
        }
    }
}
