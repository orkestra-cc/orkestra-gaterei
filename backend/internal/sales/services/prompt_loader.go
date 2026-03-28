package services

import (
	"bytes"
	"fmt"
	"io/fs"
	"log/slog"
	"sync"
	"text/template"

	"github.com/orkestra/backend/internal/sales/prompts"
)

// PromptLoader loads and caches markdown prompt templates with locale fallback.
// Prompts are embedded into the binary via Go's embed package.
type PromptLoader struct {
	fs     fs.FS
	cache  map[string]*template.Template
	mu     sync.RWMutex
	logger *slog.Logger
}

// NewPromptLoader creates a PromptLoader backed by the embedded prompt files.
// The basePath parameter is ignored (kept for API compatibility) — prompts are always embedded.
func NewPromptLoader(_ string, logger *slog.Logger) *PromptLoader {
	return &PromptLoader{
		fs:     prompts.FS,
		cache:  make(map[string]*template.Template),
		logger: logger.With(slog.String("component", "prompt-loader")),
	}
}

// Load returns a rendered prompt for the given name and locale.
// It tries locale-specific first (locale/{locale}/{name}.md), then falls back
// to the category path (agents/{name}.md or skills/{name}.md).
func (l *PromptLoader) Load(category, name, locale string, vars map[string]any) (string, error) {
	paths := []string{}
	if locale != "" {
		paths = append(paths, "locale/"+locale+"/"+name+".md")
	}
	paths = append(paths, category+"/"+name+".md")

	var tmpl *template.Template
	var err error

	for _, path := range paths {
		tmpl, err = l.getOrLoad(path)
		if err == nil {
			break
		}
	}
	if tmpl == nil {
		return "", fmt.Errorf("prompt not found: %s/%s (locale=%s): %w", category, name, locale, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", fmt.Errorf("prompt template execution failed for %s: %w", name, err)
	}
	return buf.String(), nil
}

// LoadSkill is a convenience wrapper for loading skill prompts
func (l *PromptLoader) LoadSkill(name, locale string, vars map[string]any) (string, error) {
	return l.Load("skills", name, locale, vars)
}

// LoadAgent is a convenience wrapper for loading agent prompts
func (l *PromptLoader) LoadAgent(name, locale string, vars map[string]any) (string, error) {
	return l.Load("agents", name, locale, vars)
}

func (l *PromptLoader) getOrLoad(path string) (*template.Template, error) {
	l.mu.RLock()
	if tmpl, ok := l.cache[path]; ok {
		l.mu.RUnlock()
		return tmpl, nil
	}
	l.mu.RUnlock()

	data, err := fs.ReadFile(l.fs, path)
	if err != nil {
		return nil, err
	}

	tmpl, err := template.New(path).Parse(string(data))
	if err != nil {
		return nil, fmt.Errorf("parse template %s: %w", path, err)
	}

	l.mu.Lock()
	l.cache[path] = tmpl
	l.mu.Unlock()

	l.logger.Debug("loaded prompt template", slog.String("path", path))
	return tmpl, nil
}
