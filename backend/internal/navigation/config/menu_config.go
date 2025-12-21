package config

import (
	"os"

	"github.com/orkestra/backend/internal/navigation/models"
)

// MenuConfig holds all navigation menu definitions
type MenuConfig struct {
	groups []models.RouteGroup
}

// NewMenuConfig creates the application menu configuration
func NewMenuConfig() *MenuConfig {
	groups := []models.RouteGroup{
		buildSuperAdminRoutes(),
		buildAdminRoutes(),
		buildOperatorRoutes(),
	}

	// Add development routes only in development mode
	if isDevelopment() {
		groups = append(groups, buildDevelopmentRoutes())
	}

	return &MenuConfig{
		groups: groups,
	}
}

// GetGroups returns all menu groups
func (m *MenuConfig) GetGroups() []models.RouteGroup {
	return m.groups
}

// isDevelopment checks if running in development mode
func isDevelopment() bool {
	env := os.Getenv("APP_ENV")
	return env == "" || env == "development" || env == "dev"
}

// buildOperatorRoutes creates operator-level navigation
// Accessible by: operator, manager, administrator, ceo, developer
func buildOperatorRoutes() models.RouteGroup {
	return models.RouteGroup{
		Label: "Operators",
		Roles: []string{"operator"},
		Children: []models.NavItem{
			{
				Name:   "Dashboard",
				Icon:   "chart-pie",
				To:     "/user/dashboard",
				Active: true,
				Exact:  true,
				Roles:  []string{"operator"},
			},
			{
				Name:   "Profile",
				Icon:   "user",
				To:     "/user/profile",
				Active: true,
				Roles:  []string{"operator"},
			},
			{
				Name:   "Calendar",
				Icon:   "calendar-alt",
				To:     "/user/calendar",
				Active: true,
				Roles:  []string{"operator"},
			},
		},
	}
}

// buildAdminRoutes creates administrator-level navigation
// Accessible by: administrator, ceo, developer
func buildAdminRoutes() models.RouteGroup {
	return models.RouteGroup{
		Label: "Administration",
		Roles: []string{"administrator"},
		Children: []models.NavItem{
			{
				Name:   "Fleet Management",
				Icon:   "truck",
				Active: true,
				Roles:  []string{"administrator"},
				Children: []models.NavItem{
					{
						Name:   "Vehicles",
						To:     "/fleet/vehicles",
						Active: true,
						Roles:  []string{"administrator"},
					},
					{
						Name:   "Cranes",
						To:     "/fleet/cranes",
						Active: true,
						Roles:  []string{"administrator"},
					},
					{
						Name:   "Tachographs",
						To:     "/fleet/tachographs",
						Active: true,
						Roles:  []string{"administrator"},
					},
				},
			},
			{
				Name:   "Deadlines",
				To:     "/reports/deadlines",
				Icon:   "calendar-check",
				Active: true,
				Roles:  []string{"manager"},
			},
		},
	}
}

// buildSuperAdminRoutes creates super administrator system management navigation
// Accessible by: administrator, ceo, developer
func buildSuperAdminRoutes() models.RouteGroup {
	return models.RouteGroup{
		Label: "System Administration",
		Roles: []string{"administrator"},
		Children: []models.NavItem{
			{
				Name:   "User Management",
				Icon:   "users",
				To:     "/admin/users",
				Active: true,
				Roles:  []string{"administrator"},
			},
			{
				Name:   "Settings",
				Icon:   "cog",
				To:     "/admin/settings",
				Active: true,
				Roles:  []string{"administrator"},
			},
		},
	}
}

// buildDevelopmentRoutes creates development-only navigation
// Accessible by: developer only
func buildDevelopmentRoutes() models.RouteGroup {
	return models.RouteGroup{
		Label: "Development",
		Roles: []string{"developer"},
		Children: []models.NavItem{
			{
				Name:   "Dashboard",
				Icon:   "chart-pie",
				Active: true,
				Roles:  []string{"developer"},
				Children: []models.NavItem{
					{
						Name:   "Default",
						To:     "/",
						Active: true,
						Exact:  true,
						Roles:  []string{"developer"},
					},
					{
						Name:   "Analytics",
						To:     "/dashboard/analytics",
						Active: true,
						Roles:  []string{"developer"},
					},
					{
						Name:   "CRM",
						To:     "/dashboard/crm",
						Active: true,
						Roles:  []string{"developer"},
					},
					{
						Name:   "Management",
						To:     "/dashboard/project-management",
						Active: true,
						Roles:  []string{"developer"},
					},
					{
						Name:   "SaaS",
						To:     "/dashboard/saas",
						Active: true,
						Roles:  []string{"developer"},
					},
					{
						Name:   "Support Desk",
						To:     "/dashboard/support-desk",
						Active: true,
						Roles:  []string{"developer"},
					},
				},
			},
			{
				Name:   "Applications",
				Icon:   "th",
				Active: true,
				Roles:  []string{"developer"},
				Children: []models.NavItem{
					{
						Name:   "Calendar",
						To:     "/app/calendar",
						Active: true,
						Roles:  []string{"developer"},
					},
					{
						Name:   "Chat",
						To:     "/app/chat",
						Active: true,
						Roles:  []string{"developer"},
					},
					{
						Name:   "Kanban",
						To:     "/app/kanban",
						Active: true,
						Roles:  []string{"developer"},
					},
				},
			},
			{
				Name:   "Components",
				Icon:   "puzzle-piece",
				Active: true,
				Roles:  []string{"developer"},
				Children: []models.NavItem{
					{
						Name:   "Accordion",
						To:     "/components/accordion",
						Active: true,
						Roles:  []string{"developer"},
					},
					{
						Name:   "Alerts",
						To:     "/components/alerts",
						Active: true,
						Roles:  []string{"developer"},
					},
					{
						Name:   "Buttons",
						To:     "/components/buttons",
						Active: true,
						Roles:  []string{"developer"},
					},
					{
						Name:   "Cards",
						To:     "/components/cards",
						Active: true,
						Roles:  []string{"developer"},
					},
					{
						Name:   "Modals",
						To:     "/components/modals",
						Active: true,
						Roles:  []string{"developer"},
					},
				},
			},
			{
				Name:   "Documentation",
				Icon:   "book",
				To:     "/documentation/getting-started",
				Active: true,
				Roles:  []string{"developer"},
			},
		},
	}
}
