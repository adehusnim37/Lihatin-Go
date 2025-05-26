package repositories

import (
	"database/sql"
	"log"
	"time"

	"github.com/adehusnim37/lihatin-go/models"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/google/uuid"
)

// UserRepository defines the methods for user-related database operations
type UserRepository interface {
	GetAllUsers() ([]models.User, error)
	GetUserByID(id string) (*models.User, error)
	GetUserByEmailOrUsername(input string) (*models.User, error)
	GetUserForLogin(emailOrUsername string) (*models.User, error)
	CheckPremiumByUsernameOrEmail(inputs string) (*models.User, error)
	CreateUser(user *models.User) error
	UpdateUser(id string, user *models.User) error
	DeleteUserPermanent(id string) error
}

type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepository{
		db: db,
	}
}

func (ur *userRepository) GetAllUsers() ([]models.User, error) {
	rows, err := ur.db.Query("SELECT id, first_name, last_name, email, password, created_at, updated_at, deleted_at, is_premium, avatar, username FROM users")
	if err != nil {
		log.Printf(`Error executing query: %v`, err)
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User

		if err := rows.Scan(&user.ID, &user.FirstName, &user.LastName, &user.Email, &user.Password,
			&user.CreatedAt, &user.UpdatedAt, &user.DeletedAt, &user.IsPremium, &user.Avatar, &user.Username); err != nil {
			log.Printf(`Error scanning user: %v`, err)
			return nil, err
		}

		users = append(users, user)
	}

	return users, nil
}

func (ur *userRepository) GetUserByID(id string) (*models.User, error) {
	// First log the query we're about to execute for debugging
	log.Printf("GetUserByID: Executing query for ID: %s", id)

	row := ur.db.QueryRow("SELECT id, first_name, last_name, email, password, created_at, updated_at, deleted_at, is_premium, avatar, username FROM users WHERE id = ?", id)

	var user models.User

	if err := row.Scan(&user.ID, &user.FirstName, &user.LastName, &user.Email, &user.Password,
		&user.CreatedAt, &user.UpdatedAt, &user.DeletedAt, &user.IsPremium, &user.Avatar, &user.Username); err != nil {
		log.Printf("Error scanning user by ID %s: %v", id, err)
		return nil, err
	}

	log.Printf("User found: ID=%s, Username=%s", user.ID, user.Username)
	return &user, nil
}

func (ur *userRepository) CheckPremiumByUsernameOrEmail(inputs string) (*models.User, error) {
	row := ur.db.QueryRow("SELECT is_premium FROM users WHERE username = ? OR email = ?", inputs, inputs)

	var user models.User
	if err := row.Scan(&user.IsPremium); err != nil {
		if err == sql.ErrNoRows {
			// No user found with this username/email
			return nil, err
		}
		// Other database error
		return nil, err
	}
	return &user, nil
}

func (ur *userRepository) GetUserByEmailOrUsername(input string) (*models.User, error) {
	// Check if it's an email or username
	row := ur.db.QueryRow("SELECT id, email, username FROM users WHERE email = ? OR username = ?", input, input)

	var user models.User
	err := row.Scan(&user.ID, &user.Email, &user.Username)
	if err != nil {
		if err == sql.ErrNoRows {
			// No user found with this email/username
			return nil, err
		}
		// Other database error
		return nil, err
	}

	// User found
	return &user, nil
}

func (ur *userRepository) CreateUser(user *models.User) error {
	// Hash the password before storing
	hashedPassword, err := utils.HashPassword(user.Password)
	if err != nil {
		log.Printf("CreateUser: Error hashing password: %v", err)
		return err
	}

	now := time.Now()
	u := uuid.Must(uuid.NewRandom())
	_, err = ur.db.Exec("INSERT INTO users (id, first_name, last_name, email, password, created_at, updated_at, deleted_at, is_premium, avatar, username) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		u, user.FirstName, user.LastName, user.Email, hashedPassword, now, now, nil, user.IsPremium, user.Avatar, user.Username)
	if err != nil {
		log.Printf("CreateUser error: %v", err)
	}
	return err
}

func (ur *userRepository) UpdateUser(id string, user *models.User) error {
	// Hash password if it's being updated
	hashedPassword := user.Password
	if user.Password != "" {
		var err error
		hashedPassword, err = utils.HashPassword(user.Password)
		if err != nil {
			log.Printf("UpdateUser: Error hashing password: %v", err)
			return err
		}
	}

	_, err := ur.db.Exec("UPDATE users SET first_name = ?, last_name = ?, email = ?, password = ?, updated_at = ?, is_premium = ?, avatar = ? WHERE id = ?",
		user.FirstName, user.LastName, user.Email, hashedPassword, time.Now(), user.IsPremium, user.Avatar, id)
	return err
}

func (ur *userRepository) GetUserForLogin(emailOrUsername string) (*models.User, error) {
	log.Printf("GetUserForLogin: Attempting login for: %s", emailOrUsername)

	row := ur.db.QueryRow(`
		SELECT id, username, first_name, last_name, email, password, 
		       created_at, updated_at, deleted_at, is_premium, avatar 
		FROM users 
		WHERE (email = ? OR username = ?) AND deleted_at IS NULL`,
		emailOrUsername, emailOrUsername)

	var user models.User
	if err := row.Scan(&user.ID, &user.Username, &user.FirstName, &user.LastName,
		&user.Email, &user.Password, &user.CreatedAt, &user.UpdatedAt,
		&user.DeletedAt, &user.IsPremium, &user.Avatar); err != nil {

		if err == sql.ErrNoRows {
			log.Printf("No user found with email/username: %s", emailOrUsername)
			return nil, err
		}
		log.Printf("Error scanning user for login: %v", err)
		return nil, err
	}

	log.Printf("User found for login: ID=%s, Username=%s", user.ID, user.Username)
	return &user, nil
}

func (ur *userRepository) DeleteUserPermanent(id string) error {
	_, err := ur.db.Exec("DELETE FROM users WHERE id = ?", id)
	return err
}
