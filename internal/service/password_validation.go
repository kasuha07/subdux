package service

const bcryptMaxPasswordBytes = 72

func validateBcryptPasswordLength(password string) error {
	if len([]byte(password)) > bcryptMaxPasswordBytes {
		return ErrPasswordTooLong
	}
	return nil
}
