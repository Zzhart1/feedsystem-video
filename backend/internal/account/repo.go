package account

import (
	"context"

	"gorm.io/gorm"
)

type AccountRepository struct{
	db *gorm.DB
}

func NewAccountRepository(db *gorm.DB) *AccountRepository {
	return &AccountRepository{db:db}
}

func (ar *AccountRepository) CreateAccount(ctx context.Context,account *Account) error{
	if err:=ar.db.WithContext(ctx).Create(account).Error;err!=nil{
		return err
	}
	return nil
}

func (ar *AccountRepository) Rename(ctx context.Context, id uint, newUsername string) error {
	if err:=ar.db.WithContext(ctx).Model(&Account{}).Where("id=?",id).Update("username",newUsername).Error;err!=nil{
		return err
	}
	return nil
}

func (ar *AccountRepository) RenameWithToken(ctx context.Context, id uint, newUsername string, token string) error {
	//为什么要带着token去更新用户名呢？因为我们需要确保这个请求是合法的，只有拥有正确token的用户才能修改自己的用户名。
	if err:=ar.db.WithContext(ctx).Model(&Account{}).Where("id=? AND token=?",id,token).Update("username",newUsername).Error;err!=nil{
		return err
	}
	return nil
}

func (ar *AccountRepository) ChangePassword(ctx context.Context, username string, oldPassword string, newPassword string) error {
	if err:=ar.db.WithContext(ctx).Model(&Account{}).Where("username=? AND password=?",username,oldPassword).Update("password",newPassword).Error;err!=nil{
		return err
	}
	return nil
}

func (ar *AccountRepository) FindByID(ctx context.Context, id uint) (*Account, error) {
	var account Account
	if err:=ar.db.WithContext(ctx).First(&account,id).Error;err!=nil{
		return nil,err
	}
	return &account,nil
}

func (ar *AccountRepository) FindByUsername(ctx context.Context, username string) (*Account, error) {
	var account Account
	if err:=ar.db.WithContext(ctx).Where("username=?",username).First(&account).Error;err!=nil{
		return nil,err
	}
	return &account,nil
}

func (ar *AccountRepository) Login(ctx context.Context, id uint,token,refreshToken string) error {
	if err:=ar.db.WithContext(ctx).Model(&Account{}).Where("id=?",id).Updates(map[string]interface{}{
		"token":token,
		"refresh_token":refreshToken,
	}).Error;err!=nil{
		return err
	}
	return nil
}

func (ar *AccountRepository) Logout(ctx context.Context, id uint) error {
	if err:=ar.db.WithContext(ctx).Model(&Account{}).Where("id=?",id).Updates(map[string]interface{}{
		"token":"",
		"refresh_token":"",
	}).Error;err!=nil{
		return err
	}
	return nil
}

func (ar *AccountRepository) UpdateAvatar(ctx context.Context, id uint, avatarURL string) error {
	if err:=ar.db.WithContext(ctx).Model(&Account{}).Where("id=?",id).Update("avatar_url",avatarURL).Error;err!=nil{
		return err
	}
	return nil
}

func (ar *AccountRepository) UpdateToken(ctx context.Context, id uint, token string) error {
	if err:=ar.db.WithContext(ctx).Model(&Account{}).Where("id=?",id).Update("token",token).Error;err!=nil{
		return err
	}
	return nil
}

func (ar *AccountRepository) UpdateFields(ctx context.Context, id uint,fields map[string]interface{}) error {
	if err:=ar.db.WithContext(ctx).Model(&Account{}).Where("id=?",id).Updates(fields).Error;err!=nil{
		return err
	}
	return nil
}

func (ar *AccountRepository) FindAll(ctx context.Context) ([]*Account, error) {
	var accounts []*Account
	if err:=ar.db.WithContext(ctx).Find(&accounts).Error;err!=nil{
		return nil,err
	}
	return accounts,nil
}