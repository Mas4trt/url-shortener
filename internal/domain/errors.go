package domain

import "errors"

var (
	ErrURLNotFound = errors.New("url not found")
	ErrURLExist    = errors.New("url already exist")
	ErrEmptyAlias  = errors.New("empty alias")
	ErrInvalidURL  = errors.New("empty url")
)
