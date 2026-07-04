package service

import serviceauth "github.com/kasuha07/subdux/internal/service/auth"

const bcryptMaxPasswordBytes = 72

func validateBcryptPasswordLength(password string) error {
	if len([]byte(password)) > bcryptMaxPasswordBytes {
		return serviceauth.ErrPasswordTooLong
	}
	return nil
}
