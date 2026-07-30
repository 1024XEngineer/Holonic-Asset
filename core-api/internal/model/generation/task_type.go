package generation

type TaskType string

const (
	GenerateCharacterProtoType   TaskType = "generate_character_prototype"
	GenerateCharacterAnimation   TaskType = "generate_character_animation"
	RegenerateCharacterProtoType TaskType = "regenerate_character_prototype"
	RegenerateCharacterAnimation TaskType = "regenerate_character_animation"
	RegenerateCharacterFrames    TaskType = "regenerate_character_frames"

	GenerateObjectProtoType   TaskType = "generate_object_prototype"
	GenerateObjectAnimation   TaskType = "generate_object_animation"
	RegenerateObjectProtoType TaskType = "regenerate_object_prototype"
	RegenerateObjectAnimation TaskType = "regenerate_object_animation"
	RegenerateObjectFrames    TaskType = "regenerate_object_frames"

	GenerateTileSet TaskType = "generate_tileset"
	RegenerateItem  TaskType = "regenerate_item"
	RegenerateTiles TaskType = "regenerate_tiles"
)

func ProjectLevelTaskTypes() []TaskType {
	return []TaskType{
		GenerateCharacterProtoType,
		GenerateObjectProtoType,
		GenerateTileSet,
	}
}

func TaskTypes() []TaskType {
	return []TaskType{
		GenerateCharacterProtoType,
		GenerateCharacterAnimation,
		RegenerateCharacterProtoType,
		RegenerateCharacterAnimation,
		RegenerateCharacterFrames,
		GenerateObjectProtoType,
		GenerateObjectAnimation,
		RegenerateObjectProtoType,
		RegenerateObjectAnimation,
		RegenerateObjectFrames,
		GenerateTileSet,
		RegenerateItem,
		RegenerateTiles,
	}
}
