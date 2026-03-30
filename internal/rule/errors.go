// SPDX-FileCopyrightText: Copyright 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package rule

import (
	"errors"
	"fmt"
)

var (
	ErrEmptyRule = errors.New("empty hostname rule")
)

// ErrKeyValFormat represents an error that occurs when a hostname rule element
// does not follow the key:val format.
type ErrKeyValFormat struct {
	Elem int
	Got  string
}

func NewErrKeyValFormat(elem int, got string) ErrKeyValFormat {
	return ErrKeyValFormat{
		Elem: elem,
		Got:  got,
	}
}

func (ekvf ErrKeyValFormat) Error() string {
	return fmt.Sprintf("element %d: expected key:val, got %q", ekvf.Elem, ekvf.Got)
}

// ErrNoKey represents an error that occurs when a rule has no key (e.g.
// ':val').
type ErrNoKey struct {
	Elem int
	Got  string
}

func NewErrNoKey(elem int, got string) ErrNoKey {
	return ErrNoKey{
		Elem: elem,
		Got:  got,
	}
}

func (enk ErrNoKey) Error() string {
	return fmt.Sprintf("element %d: empty key (got %q)", enk.Elem, enk.Got)
}

// ErrUnknownKey represents an error that occurs when a rule contains an unknown
// key.
type ErrUnknownKey struct {
	Elem int
	Key  string
}

func NewErrUnknownKey(elem int, key string) ErrUnknownKey {
	return ErrUnknownKey{
		Elem: elem,
		Key:  key,
	}
}

func (euk ErrUnknownKey) Error() string {
	return fmt.Sprintf("element %d: unknown key %q", euk.Elem, euk.Key)
}

// ErrBadQuote represents an error that occurs when there is a quoting error in
// a hostname rule.
type ErrBadQuote struct {
	Elem     int
	Got      string
	QuoteErr error
}

func NewErrBadQuote(elem int, got string, quoteErr error) ErrBadQuote {
	return ErrBadQuote{
		Elem:     elem,
		Got:      got,
		QuoteErr: quoteErr,
	}
}

func (ebq ErrBadQuote) Error() string {
	return fmt.Sprintf("element %d: bad quoting: %v: got %q", ebq.Elem, ebq.QuoteErr, ebq.Got)
}

// ErrDuplicateKey represents an error that occurs when a duplicate key is
// specified.
type ErrDuplicateKey struct {
	Elem int
	Got  string
	Key  string
}

func NewErrDuplicateKey(elem int, got, key string) ErrDuplicateKey {
	return ErrDuplicateKey{
		Elem: elem,
		Got:  got,
		Key:  key,
	}
}

func (edk ErrDuplicateKey) Error() string {
	return fmt.Sprintf("element %d: duplicate key %q: got %q", edk.Elem, edk.Key, edk.Got)
}

// ErrRequiredKeys represents an error that occurs when one or more required
// keys are not specified.
type ErrRequiredKeys struct {
	Keys []string
}

func NewErrRequiredKeys(keys ...string) ErrRequiredKeys {
	return ErrRequiredKeys{
		Keys: keys,
	}
}

func (erk ErrRequiredKeys) Error() string {
	return fmt.Sprintf("required key missing, at least one of %v", erk.Keys)
}

// ErrInvalidValue represents an error where a key has an unexpected value. A
// string representing the expected value is stored as well to be printed in the
// error.
type ErrInvalidValue struct {
	Key      string
	Value    string
	Expected string
}

func NewErrInvalidValue(key, value, expected string) ErrInvalidValue {
	return ErrInvalidValue{
		Key:      key,
		Value:    value,
		Expected: expected,
	}
}

func (eiv ErrInvalidValue) Error() string {
	return fmt.Sprintf("invalid value for key %q (expected %s but got %q)", eiv.Key, eiv.Expected, eiv.Value)
}

// ErrMutualExclusion represents an error where two or more mutually-exclusive
// keys were specified.
type ErrMutualExclusion struct {
	Keys []string
}

func NewErrMutualExclusion(keys ...string) ErrMutualExclusion {
	return ErrMutualExclusion{
		Keys: keys,
	}
}

func (eme ErrMutualExclusion) Error() string {
	return fmt.Sprintf("keys are mutually exclusive: %v", eme.Keys)
}
