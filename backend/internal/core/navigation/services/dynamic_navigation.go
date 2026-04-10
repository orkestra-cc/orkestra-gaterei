package services

import (
	"context"

	"github.com/orkestra/backend/internal/core/navigation/models"
	"github.com/orkestra/backend/internal/shared/middleware"
	"github.com/orkestra/backend/internal/shared/module"
)

// dynamicNavigationService builds navigation from module-declared NavItems
// and filters by module enabled status. Permission-based menu filtering is
// delegated to the frontend, which consults the authz provider's effective
// permissions response to hide items the user cannot access.
type dynamicNavigationService struct {
	navItems       []module.NavItemSpec
	enabledChecker middleware.ModuleEnabledChecker
}

// NewDynamicNavigationService creates a navigation service that derives its
// menu from module NavItemSpec declarations instead of a hardcoded config.
func NewDynamicNavigationService(items []module.NavItemSpec, checker middleware.ModuleEnabledChecker) NavigationService {
	return &dynamicNavigationService{
		navItems:       items,
		enabledChecker: checker,
	}
}

func (s *dynamicNavigationService) GetNavigationForUser(ctx context.Context, userRole string) (*models.NavigationResponse, error) {
	groupMap := make(map[string][]module.NavItemSpec)
	var groupOrder []string
	for _, item := range s.navItems {
		group := item.Group
		if group == "" {
			group = "Other"
		}
		if _, exists := groupMap[group]; !exists {
			groupOrder = append(groupOrder, group)
		}
		groupMap[group] = append(groupMap[group], item)
	}

	var groups []models.RouteGroup
	for _, groupLabel := range groupOrder {
		items := groupMap[groupLabel]
		children := s.filterAndConvert(ctx, items)
		if len(children) > 0 {
			groups = append(groups, models.RouteGroup{
				Label:    groupLabel,
				Children: children,
			})
		}
	}

	return &models.NavigationResponse{
		Groups:    groups,
		UserRole:  userRole,
		CacheKey:  "nav:" + userRole,
		ExpiresIn: 300,
	}, nil
}

// filterAndConvert converts NavItemSpecs to NavItems, filtering disabled modules.
func (s *dynamicNavigationService) filterAndConvert(ctx context.Context, specs []module.NavItemSpec) []models.NavItem {
	var result []models.NavItem
	for _, spec := range specs {
		if spec.ModuleName != "" && s.enabledChecker != nil {
			if !s.enabledChecker.IsEnabled(ctx, spec.ModuleName) {
				continue
			}
		}

		item := models.NavItem{
			Name:   spec.Name,
			To:     spec.Path,
			Icon:   spec.Icon,
			Active: spec.Active,
		}

		if len(spec.Children) > 0 {
			item.Children = s.filterAndConvert(ctx, spec.Children)
			if len(item.Children) == 0 && item.To == "" {
				continue
			}
		}

		result = append(result, item)
	}
	return result
}
