package commands

type UserCancelError struct{}

func (e *UserCancelError) Error() string {
	return "the user canceled the request"
}
