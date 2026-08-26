package tag

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	DefaultTagColor         = "#4F46E5"
	maxTagNameLength        = 100
	maxTagDescriptionLength = 255
)

var (
	ErrInvalidTag         = errors.New("invalid project tag")
	ErrTagNotFound        = errors.New("project tag not found")
	ErrTagConflict        = errors.New("project tag already exists")
	ErrTagProjectNotFound = errors.New("project not found")
	tagColorPattern       = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
)

// Tag is reusable metadata scoped to one project and associated with assets.
type Tag struct {
	ID          uint
	ProjectID   uint
	Name        string
	Description string
	Color       string
}

// TagUpdate contains the fields that may be changed on a project tag.
type TagUpdate struct {
	Name        *string
	Description *string
	Color       *string
}

func (t *Tag) validateCreate() error {
	if t == nil {
		return invalidTag("tag is required")
	}
	if t.ProjectID == 0 {
		return invalidTag("projectID is required")
	}
	name, err := normalizeTagName(t.Name)
	if err != nil {
		return err
	}
	description, err := normalizeTagDescription(t.Description)
	if err != nil {
		return err
	}
	color := strings.TrimSpace(t.Color)
	if color == "" {
		color = DefaultTagColor
	}
	if err := validateTagColor(color); err != nil {
		return err
	}
	t.Name = name
	t.Description = description
	t.Color = color
	return nil
}

func (u *TagUpdate) validate() error {
	if u == nil {
		return invalidTag("tag update is required")
	}
	if u.Name == nil && u.Description == nil && u.Color == nil {
		return invalidTag("at least one update field is required")
	}
	if u.Name != nil {
		value, err := normalizeTagName(*u.Name)
		if err != nil {
			return err
		}
		u.Name = &value
	}
	if u.Description != nil {
		value, err := normalizeTagDescription(*u.Description)
		if err != nil {
			return err
		}
		u.Description = &value
	}
	if u.Color != nil {
		value := strings.TrimSpace(*u.Color)
		if err := validateTagColor(value); err != nil {
			return err
		}
		u.Color = &value
	}
	return nil
}

func normalizeTagName(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", invalidTag("name is required")
	}
	if utf8.RuneCountInString(value) > maxTagNameLength {
		return "", invalidTag("name exceeds maximum length of 100 characters")
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return "", invalidTag("name contains invalid control characters")
		}
	}
	return value, nil
}

func normalizeTagDescription(value string) (string, error) {
	value = strings.TrimSpace(value)
	if utf8.RuneCountInString(value) > maxTagDescriptionLength {
		return "", invalidTag("description exceeds maximum length of 255 characters")
	}
	return value, nil
}

func validateTagColor(value string) error {
	if !tagColorPattern.MatchString(value) {
		return invalidTag("color must use #RRGGBB format")
	}
	return nil
}

func validateTagScope(projectID, tagID uint) error {
	if projectID == 0 {
		return invalidTag("projectID is required")
	}
	if tagID == 0 {
		return invalidTag("tagID is required")
	}
	return nil
}

func invalidTag(reason string) error {
	return fmt.Errorf("%w: %s", ErrInvalidTag, reason)
}
