package auth

import "fyp/food-rs/types/model"

func (s *service) generateToken(user *model.User) (string, error) {
	// TODO
	return user.ID.String(), nil
}
