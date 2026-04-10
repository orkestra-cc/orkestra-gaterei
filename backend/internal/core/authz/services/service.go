package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/orkestra/backend/internal/core/authz/models"
	"github.com/orkestra/backend/internal/core/authz/repository"
	"github.com/orkestra/backend/internal/shared/database"
	"github.com/orkestra/backend/internal/shared/iface"
)

// Service owns authorization lifecycle and implements iface.AuthzProvider.
//
// Permission evaluation rules (in order):
//  1. If the user's system role is "developer", every permission is granted.
//  2. If the user's system role is "administrator", every system permission
//     and every non-delete permission is granted (customizable later).
//  3. Otherwise, the user's permissions in the given org are the union of
//     all permissions from non-expired role bindings for (userUUID, orgID),
//     plus any bindings on the user with orgID="" (global grants).
//  4. System permissions (where PermissionSpec.System is true) require the
//     user to have the permission granted globally (either by system role
//     or by a global binding).
//
// Results are cached in Redis for 60 seconds per (userUUID, orgID) key and
// invalidated when bindings or roles change.
type Service struct {
	repo      *repository.Repository
	redis     *database.RedisClientAdapter
	logger    *slog.Logger
	userRoles UserSystemRoleLookup

	mu                  sync.RWMutex
	systemPermissionSet map[string]struct{} // keys declared with System=true
	allPermissionSet    map[string]struct{} // every registered permission
}

// UserSystemRoleLookup resolves a user's system role (from the user module).
// Kept as a plain function type so we don't need to import the user module.
type UserSystemRoleLookup func(ctx context.Context, userUUID string) (string, error)

type Config struct {
	Repo       *repository.Repository
	Redis      *database.RedisClientAdapter
	Logger     *slog.Logger
	LookupUser UserSystemRoleLookup
}

func New(cfg Config) *Service {
	return &Service{
		repo:                cfg.Repo,
		redis:               cfg.Redis,
		logger:              cfg.Logger,
		userRoles:           cfg.LookupUser,
		systemPermissionSet: make(map[string]struct{}),
		allPermissionSet:    make(map[string]struct{}),
	}
}

// --- Provider interface ---

func (s *Service) HasPermission(ctx context.Context, userUUID, orgID, permission string) (bool, error) {
	perms, err := s.GetEffectivePermissions(ctx, userUUID, orgID)
	if err != nil {
		return false, err
	}
	for _, p := range perms {
		if p == permission || p == "*" {
			return true, nil
		}
	}
	return false, nil
}

func (s *Service) GetEffectivePermissions(ctx context.Context, userUUID, orgID string) ([]string, error) {
	if userUUID == "" {
		return nil, errors.New("authz: userUUID required")
	}

	if cached, ok := s.cacheGet(ctx, userUUID, orgID); ok {
		return cached, nil
	}

	systemRole := ""
	if s.userRoles != nil {
		r, err := s.userRoles(ctx, userUUID)
		if err == nil {
			systemRole = r
		}
	}

	perms := make(map[string]struct{})

	// System role: developer gets *, administrator gets all system perms.
	switch systemRole {
	case "developer":
		perms["*"] = struct{}{}
	case "administrator":
		s.mu.RLock()
		for k := range s.systemPermissionSet {
			perms[k] = struct{}{}
		}
		s.mu.RUnlock()
	}

	// Union of global bindings (orgID="").
	globals, err := s.repo.ListActiveBindingsForUser(ctx, userUUID, "")
	if err != nil {
		return nil, err
	}
	for _, b := range globals {
		role, err := s.repo.GetRoleByUUID(ctx, b.RoleUUID)
		if err != nil {
			continue
		}
		for _, p := range role.Permissions {
			perms[p] = struct{}{}
		}
	}

	// Union of org-scoped bindings.
	if orgID != "" {
		scoped, err := s.repo.ListActiveBindingsForUser(ctx, userUUID, orgID)
		if err != nil {
			return nil, err
		}
		for _, b := range scoped {
			role, err := s.repo.GetRoleByUUID(ctx, b.RoleUUID)
			if err != nil {
				continue
			}
			for _, p := range role.Permissions {
				perms[p] = struct{}{}
			}
		}
	}

	out := make([]string, 0, len(perms))
	for k := range perms {
		out = append(out, k)
	}
	s.cacheSet(ctx, userUUID, orgID, out)
	return out, nil
}

func (s *Service) RegisterPermissions(ctx context.Context, specs []iface.PermissionSpec) error {
	if len(specs) == 0 {
		return nil
	}
	s.mu.Lock()
	for _, spec := range specs {
		s.allPermissionSet[spec.Key] = struct{}{}
		if spec.System {
			s.systemPermissionSet[spec.Key] = struct{}{}
		}
	}
	s.mu.Unlock()

	for _, spec := range specs {
		p := &models.Permission{
			Key:         spec.Key,
			Module:      spec.Module,
			Description: spec.Description,
			System:      spec.System,
		}
		if err := s.repo.UpsertPermission(ctx, p); err != nil {
			return fmt.Errorf("upsert permission %s: %w", spec.Key, err)
		}
	}
	return nil
}

// --- System role seeding ---

// SeedSystemRoles creates the six default system roles on first boot. They
// have orgId="" and IsSystem=true. Permission lists are derived from the
// permissions catalog that modules have registered by the time this runs.
// Call this after RegisterPermissions has been called for every module.
func (s *Service) SeedSystemRoles(ctx context.Context) error {
	allKeys, err := s.repo.ListAllPermissionKeys(ctx)
	if err != nil {
		return err
	}
	systemKeys, err := s.repo.ListSystemPermissionKeys(ctx)
	if err != nil {
		return err
	}

	nonDelete := filter(allKeys, func(p string) bool {
		return !strings.HasSuffix(p, ".delete")
	})

	// Operator: read-only + self-service update permissions
	operator := filter(allKeys, func(p string) bool {
		return strings.HasSuffix(p, ".read") || strings.HasSuffix(p, ".self")
	})

	// Manager: read + create + update (no admin, no delete)
	manager := filter(allKeys, func(p string) bool {
		if strings.HasSuffix(p, ".delete") || strings.HasSuffix(p, ".admin") {
			return false
		}
		return true
	})

	// Guest: read-only
	guest := filter(allKeys, func(p string) bool {
		return strings.HasSuffix(p, ".read")
	})

	roles := []models.Role{
		{UUID: uuid.NewString(), Name: "developer", Description: "Super-admin. All permissions everywhere.", Permissions: []string{"*"}, IsSystem: true},
		{UUID: uuid.NewString(), Name: "ceo", Description: "Executive. All non-delete permissions + system permissions.", Permissions: union(nonDelete, systemKeys), IsSystem: true},
		{UUID: uuid.NewString(), Name: "administrator", Description: "Org admin. All permissions in the org.", Permissions: allKeys, IsSystem: true},
		{UUID: uuid.NewString(), Name: "manager", Description: "Read/write, no admin, no delete.", Permissions: manager, IsSystem: true},
		{UUID: uuid.NewString(), Name: "operator", Description: "Read-only + self-service.", Permissions: operator, IsSystem: true},
		{UUID: uuid.NewString(), Name: "guest", Description: "Read-only access.", Permissions: guest, IsSystem: true},
	}

	for i := range roles {
		existing, err := s.repo.GetRoleByName(ctx, "", roles[i].Name)
		if err == nil && existing != nil {
			// Preserve UUID so existing bindings keep working.
			roles[i].UUID = existing.UUID
		}
		if err := s.repo.UpsertRole(ctx, &roles[i]); err != nil {
			return fmt.Errorf("seed role %s: %w", roles[i].Name, err)
		}
	}
	return nil
}

// --- Role admin ---

func (s *Service) CreateRole(ctx context.Context, orgID string, input models.CreateRoleInput) (*models.Role, error) {
	role := &models.Role{
		UUID:        uuid.NewString(),
		OrgID:       orgID,
		Name:        input.Name,
		Description: input.Description,
		Permissions: input.Permissions,
		IsSystem:    false,
	}
	if err := s.repo.UpsertRole(ctx, role); err != nil {
		return nil, err
	}
	return role, nil
}

func (s *Service) ListRoles(ctx context.Context, orgID string) ([]models.Role, error) {
	return s.repo.ListRoles(ctx, orgID)
}

func (s *Service) DeleteRole(ctx context.Context, uuid string) error {
	return s.repo.DeleteRole(ctx, uuid)
}

func (s *Service) ListPermissions(ctx context.Context) ([]models.Permission, error) {
	return s.repo.ListPermissions(ctx)
}

// --- Bindings ---

func (s *Service) CreateBinding(ctx context.Context, orgID, grantedBy string, input models.CreateBindingInput) (*models.Binding, error) {
	role, err := s.repo.GetRoleByUUID(ctx, input.RoleUUID)
	if err != nil {
		return nil, err
	}
	b := &models.Binding{
		UUID:      uuid.NewString(),
		UserUUID:  input.UserUUID,
		OrgID:     orgID,
		RoleUUID:  role.UUID,
		RoleName:  role.Name,
		GrantedBy: grantedBy,
		ExpiresAt: input.ExpiresAt,
	}
	if err := s.repo.CreateBinding(ctx, b); err != nil {
		return nil, err
	}
	s.cacheInvalidate(ctx, input.UserUUID)
	return b, nil
}

func (s *Service) ListBindings(ctx context.Context, orgID string) ([]models.Binding, error) {
	return s.repo.ListBindingsByOrg(ctx, orgID)
}

func (s *Service) DeleteBinding(ctx context.Context, uuid string) error {
	if err := s.repo.DeleteBinding(ctx, uuid); err != nil {
		return err
	}
	s.flushCache(ctx)
	return nil
}

func (s *Service) flushCache(ctx context.Context) {
	if s.redis == nil {
		return
	}
	keys, err := s.redis.Keys(ctx, "authz:cache:*")
	if err != nil || len(keys) == 0 {
		return
	}
	_ = s.redis.Del(ctx, keys...)
}

// --- Cache ---

func (s *Service) cacheKey(userUUID, orgID string) string {
	if orgID == "" {
		orgID = "-"
	}
	return "authz:cache:" + userUUID + ":" + orgID
}

func (s *Service) cacheGet(ctx context.Context, userUUID, orgID string) ([]string, bool) {
	if s.redis == nil {
		return nil, false
	}
	raw, err := s.redis.Get(ctx, s.cacheKey(userUUID, orgID))
	if err != nil || raw == "" {
		return nil, false
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, false
	}
	return out, true
}

func (s *Service) cacheSet(ctx context.Context, userUUID, orgID string, perms []string) {
	if s.redis == nil {
		return
	}
	data, err := json.Marshal(perms)
	if err != nil {
		return
	}
	_ = s.redis.Set(ctx, s.cacheKey(userUUID, orgID), string(data), 60*time.Second)
}

func (s *Service) cacheInvalidate(ctx context.Context, userUUID string) {
	if s.redis == nil {
		return
	}
	keys, err := s.redis.Keys(ctx, "authz:cache:"+userUUID+":*")
	if err != nil || len(keys) == 0 {
		return
	}
	_ = s.redis.Del(ctx, keys...)
}

// --- helpers ---

func filter(in []string, pred func(string) bool) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if pred(s) {
			out = append(out, s)
		}
	}
	return out
}

func union(a, b []string) []string {
	set := make(map[string]struct{}, len(a)+len(b))
	for _, s := range a {
		set[s] = struct{}{}
	}
	for _, s := range b {
		set[s] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	return out
}
