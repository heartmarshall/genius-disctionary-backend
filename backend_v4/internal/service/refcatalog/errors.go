package refcatalog

import "errors"

// ErrWordNotFound indicates the word was not found by any external provider.
var ErrWordNotFound = errors.New("word not found in external provider")
