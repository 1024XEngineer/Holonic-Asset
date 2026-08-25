package asset

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	maxProjectTagNameLength        = 100
	maxProjectTagDescriptionLength = 255
)

var (
	ErrInvalidProjectTag         = errors.New("invalid project tag")
	ErrProjectTagNotFound        = errors.New("project tag not found")
	ErrProjectTagConflict        = errors.New("project tag already exists")
	ErrProjectTagProjectNotFound = errors.New("project not found")
	projectTagColorPattern       = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
)

// ProjectTag is reusable metadata scoped to one project and associated with assets.
type ProjectTag struct {
	ID          uint
	ProjectID   uint
	Name        string
	Description string
	Color       string
}

// ProjectTagUpdate contains the fields that may be changed on a project tag.
type ProjectTagUpdate struct {
	Name        *string
	Description *string
	Color       *string
}

func (t *ProjectTag) validateCreate() error {
	if t == nil {
		return invalidProjectTag("tag is required")
	}
	if t.ProjectID == 0 {
		return invalidProjectTag("projectID is required")
	}
	name, err := normalizeProjectTagName(t.Name)
	if err != nil {
		return err
	}
	description, err := normalizeProjectTagDescription(t.Description)
	if err != nil {
		return err
	}
	color := strings.TrimSpace(t.Color)
	if color == "" {
		color = DefaultTagColor
	}
	if err := validateProjectTagColor(color); err != nil {
		return err
	}
	t.Name = name
	t.Description = description
	t.Color = color
	return nil
}

func (u *ProjectTagUpdate) validate() error {
	if u == nil {
		return invalidProjectTag("tag update is required")
	}
	if u.Name == nil && u.Description == nil && u.Color == nil {
		return invalidProjectTag("at least one update field is required")
	}
	if u.Name != nil {
		value, err := normalizeProjectTagName(*u.Name)
		if err != nil {
			return err
		}
		u.Name = &value
	}
	if u.Description != nil {
		value, err := normalizeProjectTagDescription(*u.Description)
		if err != nil {
			return err
		}
		u.Description = &value
	}
	if u.Color != nil {
		value := strings.TrimSpace(*u.Color)
		if err := validateProjectTagColor(value); err != nil {
			return err
		}
		u.Color = &value
	}
	return nil
}

func normalizeProjectTagName(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", invalidProjectTag("name is required")
	}
	if utf8.RuneCountInString(value) > maxProjectTagNameLength {
		return "", invalidProjectTag("name exceeds maximum length of 100 characters")
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return "", invalidProjectTag("name contains invalid control characters")
		}
	}
	return value, nil
}

func normalizeProjectTagDescription(value string) (string, error) {
	value = strings.TrimSpace(value)
	if utf8.RuneCountInString(value) > maxProjectTagDescriptionLength {
		return "", invalidProjectTag("description exceeds maximum length of 255 characters")
	}
	return value, nil
}

func validateProjectTagColor(value string) error {
	if !projectTagColorPattern.MatchString(value) {
		return invalidProjectTag("color must use #RRGGBB format")
	}
	return nil
}

func validateProjectTagScope(projectID, tagID uint) error {
	if projectID == 0 {
		return invalidProjectTag("projectID is required")
	}
	if tagID == 0 {
		return invalidProjectTag("tagID is required")
	}
	return nil
}

func invalidProjectTag(reason string) error {
	return fmt.Errorf("%w: %s", ErrInvalidProjectTag, reason)
}
