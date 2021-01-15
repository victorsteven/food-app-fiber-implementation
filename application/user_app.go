package application

import (
	"food-app-fiber/domain/entity"
	"food-app-fiber/domain/repository"
)

type userApp struct {
	us repository.UserRepository
}

//UserApp implements the UserAppInterface
var _ UserAppInterface = &userApp{}

type UserAppInterface interface {
	SaveUser(*entity.User) (*entity.User, error)
	GetUsers() ([]entity.User, error)
	GetUser(uint64) (*entity.User, error)
	GetUserByEmailAndPassword(*entity.User) (*entity.User, error)
}

func (u *userApp) SaveUser(user *entity.User) (*entity.User, error) {
	return u.us.SaveUser(user)
}

func (u *userApp) GetUser(userId uint64) (*entity.User, error) {
	return u.us.GetUser(userId)
}

func (u *userApp) GetUsers() ([]entity.User, error) {
	return u.us.GetUsers()
}

func (u *userApp) GetUserByEmailAndPassword(user *entity.User) (*entity.User, error) {
	return u.us.GetUserByEmailAndPassword(user)
}
