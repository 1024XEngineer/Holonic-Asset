package project

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

var (
	ErrInvalidProject  = errors.New("invalid project")
	ErrProjectNotFound = errors.New("project not found")
)

func (t PlatformType) Valid() bool {
	switch t {
	case "", PlatformTypePC, PlatformTypeMobile, PlatformTypeWeb:
		return true
	default:
		return false
	}
}

func ValidateUserID(userID uint) error {
	if userID == 0 {
		return invalidProject("userID is required")
	}
	return nil
}

func ValidateProjectID(projectID uint) error {
	if projectID == 0 {
		return invalidProject("projectID is required")
	}
	return nil
}

func (p *Project) ValidateCreate() error {
	if p == nil {
		return invalidProject("project is required")
	}
	if err := ValidateUserID(p.UserID); err != nil {
		return err
	}
	return p.validateDefinition()
}

func (p *Project) ValidateReferenceGeneration() error {
	if p == nil {
		return invalidProject("project is required")
	}
	if err := p.validateDefinition(); err != nil {
		return err
	}
	if !validGenerationReference(p.Reference) {
		return invalidProject("reference must be an HTTP(S) URL")
	}
	return nil
}

func (p *Project) validateDefinition() error {
	if strings.TrimSpace(p.Name) == "" {
		return invalidProject("name is required")
	}
	if !p.Perspective.Valid() {
		return invalidProject("perspective is invalid")
	}
	if !p.TargetPlatform.Valid() {
		return invalidProject("targetPlatform is invalid")
	}
	return nil
}

func validGenerationReference(reference string) bool {
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return true
	}

	parsed, err := url.ParseRequestURI(reference)
	if err != nil || parsed.Host == "" {
		return false
	}
	return strings.EqualFold(parsed.Scheme, "http") || strings.EqualFold(parsed.Scheme, "https")
}

func (u *ProjectUpdate) Validate() error {
	if u == nil {
		return invalidProject("project update is required")
	}
	if err := ValidateProjectID(u.ID); err != nil {
		return err
	}
	if !u.hasChanges() {
		return invalidProject("at least one update field is required")
	}
	if u.Name != nil && strings.TrimSpace(*u.Name) == "" {
		return invalidProject("name is required")
	}
	if u.Perspective != nil && !u.Perspective.Valid() {
		return invalidProject("perspective is invalid")
	}
	if u.TargetPlatform != nil && !u.TargetPlatform.Valid() {
		return invalidProject("targetPlatform is invalid")
	}
	return nil
}

func (u *ProjectUpdate) hasChanges() bool {
	return u.Name != nil ||
		u.GameType != nil ||
		u.Perspective != nil ||
		u.TargetPlatform != nil ||
		u.Description != nil ||
		u.Reference != nil ||
		u.Style != nil
}

func invalidProject(reason string) error {
	return fmt.Errorf("%w: %s", ErrInvalidProject, reason)
}
