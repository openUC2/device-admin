package networkmanager

import (
	"github.com/pkg/errors"
)

// EnumInfo

type EnumInfo struct {
	Short   string
	Details string
	Level   string
}

const EnumInfoLevelError = "error"

// EnumSet

type EnumSet[T Enum] []T

type Enum interface {
	~string
	Info() EnumInfo
}

func NewEnumSet[T Enum](str []string) EnumSet[T] {
	enums := make([]T, 0, len(str))
	for _, s := range str {
		enums = append(enums, T(s))
	}
	return enums
}

func (s EnumSet[T]) Strings() []string {
	strings := make([]string, 0, len(s))
	for _, group := range s {
		strings = append(strings, string(group))
	}
	return strings
}

func (s EnumSet[T]) CheckInvalid() error {
	for _, e := range s {
		if info := e.Info(); info.Level == EnumInfoLevelError {
			return errors.New(info.Details)
		}
	}
	return nil
}
