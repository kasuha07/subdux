package auth

const bcryptMaxPasswordBytes = 72

func ValidateBcryptPasswordLength(password string) error {
	return validateBcryptPasswordLength(password)
}

func validateBcryptPasswordLength(password string) error {
	if len([]byte(password)) > bcryptMaxPasswordBytes {
		return ErrPasswordTooLong
	}
	return nil
}
