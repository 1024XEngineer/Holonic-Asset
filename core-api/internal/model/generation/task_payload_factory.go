package generation

func BuildJob(taskType TaskType, taskID, projectID uint, metadata map[string]any) any {
	aid := assetIDFromMeta(metadata)

	switch taskType {
	case GenerateCharacterProtoType:
		return GenerateCharacterProtoTypeJob{
			TaskID: taskID, ProjectID: projectID, AssetID: aid,
		}
	case GenerateCharacterAnimation:
		return GenerateCharacterAnimationJob{
			TaskID: taskID, ProjectID: projectID, AssetID: aid,
		}
	case RegenerateCharacterProtoType:
		return RegenerateCharacterProtoTypeJob{
			TaskID: taskID, ProjectID: projectID, AssetID: aid,
		}
	case RegenerateCharacterAnimation:
		return RegenerateCharacterAnimationJob{
			TaskID: taskID, ProjectID: projectID, AssetID: aid,
		}
	case RegenerateCharacterFrames:
		return RegenerateCharacterFramesJob{
			TaskID: taskID, ProjectID: projectID, AssetID: aid,
		}
	case GenerateObjectProtoType:
		return GenerateObjectProtoTypeJob{
			TaskID: taskID, ProjectID: projectID, AssetID: aid,
		}
	case GenerateObjectAnimation:
		return GenerateObjectAnimationJob{
			TaskID: taskID, ProjectID: projectID, AssetID: aid,
		}
	case RegenerateObjectProtoType:
		return RegenerateObjectProtoTypeJob{
			TaskID: taskID, ProjectID: projectID, AssetID: aid,
		}
	case RegenerateObjectAnimation:
		return RegenerateObjectAnimationJob{
			TaskID: taskID, ProjectID: projectID, AssetID: aid,
		}
	case RegenerateObjectFrames:
		return RegenerateObjectFramesJob{
			TaskID: taskID, ProjectID: projectID, AssetID: aid,
		}
	case GenerateTileSet:
		return GenerateTileSetJob{
			TaskID: taskID, ProjectID: projectID,
		}
	case RegenerateItem:
		return RegenerateItemJob{
			TaskID:    taskID,
			ProjectID: projectID,
			AssetID:   aid,
			ItemIndex: itemIndexFromMeta(metadata),
		}
	case RegenerateTiles:
		return RegenerateTilesJob{
			TaskID: taskID, ProjectID: projectID, AssetID: aid,
		}
	default:
		return nil
	}
}

func assetIDFromMeta(m map[string]any) uint {
	if m == nil {
		return 0
	}
	switch v := m["asset_id"].(type) {
	case float64:
		return uint(v)
	case uint:
		return v
	case int:
		return uint(v)
	}
	return 0
}

func itemIndexFromMeta(m map[string]any) int {
	if m == nil {
		return 0
	}
	switch v := m["item_index"].(type) {
	case float64:
		return int(v)
	case int:
		return v
	}
	return 0
}
