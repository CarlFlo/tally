package auth

import "errors"

var (
	ErrProfileNotFound          = errors.New("profile not found")
	ErrIncorrectPassword        = errors.New("incorrect password")
	ErrAuthenticationThrottled  = errors.New("please wait a moment before trying again")
	ErrPasswordTooLong          = errors.New("password is too long")
	ErrCurrentPasswordIncorrect = errors.New("current password is incorrect")
)
