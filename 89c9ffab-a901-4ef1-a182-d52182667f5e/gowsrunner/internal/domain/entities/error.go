package entities

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

type Errors struct {
	InvalidCredentials error
	UserAlreadyExists  error
	DBTemporary        error
}

var ErrorsInstance Errors = Errors{
	InvalidCredentials: errors.New("Invalid Credentials!"),
	UserAlreadyExists:  errors.New("User Already Exists!"),
	DBTemporary:        errors.New("temporary database failure"),
}


func MapPostgresError(err error) error {
	var pgError *pgconn.PgError

	if errors.As(err, &pgError) {
		switch pgError.Code {
		case "23505":
			if pgError.ConstraintName == "users_email_unique" {
				return ErrorsInstance.UserAlreadyExists
			}
			return fmt.Errorf("Unique COntraint Violation %w", err)
		case "40001", "40P01":
			return err
		}

	}
	return fmt.Errorf("db error: %w", err)
}