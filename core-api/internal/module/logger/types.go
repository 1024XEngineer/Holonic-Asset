package logger

type Logger interface {
	Debug(msg string, args ...Field)
	Info(msg string, args ...Field)
	Warn(msg string, args ...Field)
	Error(msg string, args ...Field)
}

type Field struct {
	Key string
	Val any
}

func Any(key string, val any) Field {
	return Field{
		Key: key,
		Val: val,
	}
}

func Error(err error) Field {
	return Field{
		Key: "errorx",
		Val: err,
	}
}

func Int64(key string, val int64) Field {
	return Field{
		Key: key,
		Val: val,
	}
}

func Int(key string, val int) Field {
	return Field{
		Key: key,
		Val: val,
	}
}

func String(key string, val string) Field {
	return Field{
		Key: key,
		Val: val,
	}
}

func Int32(key string, val int32) Field {
	return Field{
		Key: key,
		Val: val,
	}
}

func Float32(key string, val float32) Field {
	return Field{
		Key: key,
		Val: val,
	}
}
