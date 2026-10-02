package config

import (
	"fmt"
	"path/filepath"
	"strings"
)

// RootIdentity preserves uncertainty: unresolved is not evidence of independence.
// Unavailable roots remain valid unless their lexical paths already conflict.
type RootIdentity struct {
	Path          string
	CanonicalPath string
	ResolutionErr error
}

type RootRelation string

const (
	RootSame     RootRelation = "same"
	RootInside   RootRelation = "inside"
	RootContains RootRelation = "contains"
)

type RootConflictError struct {
	Root, Existing string
	Relation       RootRelation
	Physical       bool
}

func (e *RootConflictError) Error() string {
	suffix := ""
	if e.Physical {
		suffix = " (after resolving symlinks)"
	}
	switch e.Relation {
	case RootSame:
		return fmt.Sprintf("root %q represents the same workspace as configured root %q%s", e.Root, e.Existing, suffix)
	case RootInside:
		return fmt.Sprintf("root %q is inside configured root %q%s", e.Root, e.Existing, suffix)
	default:
		return fmt.Sprintf("root %q contains configured root %q%s", e.Root, e.Existing, suffix)
	}
}

// ValidateRoots checks lexical ownership first, then physical ownership where
// both paths resolve. Resolution failures are returned as identity diagnostics,
// not fatal configuration errors: loading must support disconnected drives.
func ValidateRoots(roots []string) ([]RootIdentity, error) {
	identities := make([]RootIdentity, 0, len(roots))
	for _, root := range roots {
		if !filepath.IsAbs(root) || root != filepath.Clean(root) {
			return identities, fmt.Errorf("root %q must be an absolute, clean path", root)
		}
		for _, existing := range identities {
			if relation := rootRelation(existing.Path, root); relation != "" {
				return identities, &RootConflictError{Root: root, Existing: existing.Path, Relation: relation}
			}
		}
		identities = append(identities, RootIdentity{Path: root})
	}
	for i := range identities {
		identity := &identities[i]
		canonical, err := filepath.EvalSymlinks(identity.Path)
		if err != nil {
			identity.ResolutionErr = fmt.Errorf("resolve root %q: %w", identity.Path, err)
			continue
		}
		identity.CanonicalPath = canonical
		for _, existing := range identities[:i] {
			if existing.ResolutionErr != nil {
				continue
			}
			if relation := rootRelation(existing.CanonicalPath, canonical); relation != "" {
				return identities, &RootConflictError{Root: identity.Path, Existing: existing.Path, Relation: relation, Physical: true}
			}
		}
	}
	return identities, nil
}

func rootRelation(existing, candidate string) RootRelation {
	if existing == candidate {
		return RootSame
	}
	inside := func(parent, child string) bool {
		rel, err := filepath.Rel(parent, child)
		return err == nil && !filepath.IsAbs(rel) && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
	}
	if inside(existing, candidate) {
		return RootInside
	}
	if inside(candidate, existing) {
		return RootContains
	}
	return ""
}
