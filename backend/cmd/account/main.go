package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"log"

	"github.com/joho/godotenv"
	"github.com/learnthor/backend/db"
	"github.com/learnthor/backend/models"
	"github.com/learnthor/backend/services"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func main() {
	username := flag.String("username", "", "existing username to grant personal premium access")
	createEmail := flag.String("create-email", "", "create a missing account with this email and a generated password")
	resetPassword := flag.Bool("reset-password", false, "replace the account password with a generated password")
	flag.Parse()
	if *username == "" {
		log.Fatal("provide --username")
	}
	_ = godotenv.Load()
	database, err := db.Connect()
	if err != nil {
		log.Fatal(err)
	}
	if err := db.Migrate(database); err != nil {
		log.Fatal(err)
	}
	password := ""
	var user *models.User
	err = database.Transaction(func(tx *gorm.DB) error {
		var existing models.User
		err := tx.Where("username = ?", *username).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) && *createEmail != "" {
			var secret [18]byte
			if _, err := rand.Read(secret[:]); err != nil {
				return err
			}
			password = hex.EncodeToString(secret[:])
			hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if err != nil {
				return err
			}
			if err := tx.Create(&models.User{Username: *username, Email: *createEmail, PasswordHash: string(hash), Role: models.UserRoleStudent, IsActive: true}).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		user, err = services.GrantPersonalPremium(tx, *username)
		if err == nil && *resetPassword && password == "" {
			var secret [18]byte
			if _, err := rand.Read(secret[:]); err != nil {
				return err
			}
			password = hex.EncodeToString(secret[:])
			hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if err != nil {
				return err
			}
			return tx.Model(&models.User{}).Where("id = ?", user.ID).Update("password_hash", string(hash)).Error
		}
		return err
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Account %s (%s): role=student, premium, personal progression bypass.\n", user.Username, user.Email)
	if password != "" {
		fmt.Printf("Generated password: %s\n", password)
	}
}
