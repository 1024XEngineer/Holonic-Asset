package repository

import (
	"bytes"
	"encoding/json"
	"fmt"
)

func migrateAnimationGeneration(currentContent, replacementContent []byte) ([]byte, error) {
	if len(currentContent) == 0 || len(replacementContent) == 0 {
		return append([]byte(nil), replacementContent...), nil
	}

	var replacement map[string]json.RawMessage
	if err := json.Unmarshal(replacementContent, &replacement); err != nil {
		return nil, fmt.Errorf("repository: decode replacement asset content: %w", err)
	}
	if replacement == nil {
		return append([]byte(nil), replacementContent...), nil
	}
	replacementAnimationsRaw, ok := replacement["animations"]
	if !ok || bytes.Equal(bytes.TrimSpace(replacementAnimationsRaw), []byte("null")) {
		return append([]byte(nil), replacementContent...), nil
	}

	var current map[string]json.RawMessage
	if err := json.Unmarshal(currentContent, &current); err != nil {
		return nil, fmt.Errorf("repository: decode current asset content: %w", err)
	}
	if current == nil {
		return append([]byte(nil), replacementContent...), nil
	}
	currentAnimationsRaw, ok := current["animations"]
	if !ok || bytes.Equal(bytes.TrimSpace(currentAnimationsRaw), []byte("null")) {
		return append([]byte(nil), replacementContent...), nil
	}

	var currentAnimations []json.RawMessage
	if err := json.Unmarshal(currentAnimationsRaw, &currentAnimations); err != nil {
		return nil, fmt.Errorf("repository: decode current asset animations: %w", err)
	}
	generationByAnimationID := make(map[uint]json.RawMessage, len(currentAnimations))
	for index, rawAnimation := range currentAnimations {
		animation, err := decodeAssetContentObject(rawAnimation)
		if err != nil {
			return nil, fmt.Errorf("repository: decode current asset animation %d: %w", index, err)
		}
		animationID, ok := decodeAssetContentID(animation["id"])
		if !ok {
			continue
		}
		generation, ok := animation["generation"]
		if !ok || len(bytes.TrimSpace(generation)) == 0 || bytes.Equal(bytes.TrimSpace(generation), []byte("null")) {
			continue
		}
		generationByAnimationID[animationID] = append(json.RawMessage(nil), generation...)
	}
	if len(generationByAnimationID) == 0 {
		return append([]byte(nil), replacementContent...), nil
	}

	var replacementAnimations []json.RawMessage
	if err := json.Unmarshal(replacementAnimationsRaw, &replacementAnimations); err != nil {
		return nil, fmt.Errorf("repository: decode replacement asset animations: %w", err)
	}
	changed := false
	for index, rawAnimation := range replacementAnimations {
		animation, err := decodeAssetContentObject(rawAnimation)
		if err != nil {
			return nil, fmt.Errorf("repository: decode replacement asset animation %d: %w", index, err)
		}
		if _, supplied := animation["generation"]; supplied {
			continue
		}
		animationID, ok := decodeAssetContentID(animation["id"])
		if !ok {
			continue
		}
		generation, ok := generationByAnimationID[animationID]
		if !ok {
			continue
		}
		animation["generation"] = append(json.RawMessage(nil), generation...)
		replacementAnimations[index], err = json.Marshal(animation)
		if err != nil {
			return nil, fmt.Errorf("repository: encode replacement asset animation %d: %w", index, err)
		}
		changed = true
	}
	if !changed {
		return append([]byte(nil), replacementContent...), nil
	}

	encodedAnimations, err := json.Marshal(replacementAnimations)
	if err != nil {
		return nil, fmt.Errorf("repository: encode replacement asset animations: %w", err)
	}
	replacement["animations"] = encodedAnimations
	encodedContent, err := json.Marshal(replacement)
	if err != nil {
		return nil, fmt.Errorf("repository: encode replacement asset content: %w", err)
	}
	return encodedContent, nil
}

func decodeAssetContentObject(raw json.RawMessage) (map[string]json.RawMessage, error) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil {
		return nil, err
	}
	if object == nil {
		return nil, fmt.Errorf("expected JSON object")
	}
	return object, nil
}

func decodeAssetContentID(raw json.RawMessage) (uint, bool) {
	if len(raw) == 0 {
		return 0, false
	}
	var id uint
	if err := json.Unmarshal(raw, &id); err != nil || id == 0 {
		return 0, false
	}
	return id, true
}
