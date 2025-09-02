package repositories

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/adehusnim37/lihatin-go/models/user"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/google/uuid"
)

// UserRepository defines the methods for user-related database operations
type UserRepository interface {
	GetAllUsers() ([]user.User, error)
	GetUserByID(id string) (*user.User, error)
	GetUserByEmailOrUsername(input string) (*user.User, error)
	GetUserForLogin(emailOrUsername string) (*user.User, error)
	CheckPremiumByUsernameOrEmail(inputs string) (*user.User, error)
	CreateUser(user *user.User) error
	UpdateUser(id string, user *user.UpdateUser) error
	DeleteUserPermanent(id string) error
	// Admin methods
	GetAllUsersWithPagination(limit, offset int) ([]user.User, int64, error)
	LockUser(userID, reason string) error
	UnlockUser(userID, reason string) error
	IsUserLocked(userID string) (bool, error)
}

type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepository{
		db: db,
	}
}

func (ur *userRepository) GetAllUsers() ([]user.User, error) {
	rows, err := ur.db.Query("SELECT id, first_name, last_name, email, password, created_at, updated_at, deleted_at, is_premium, avatar, username FROM users")
	if err != nil {
		log.Printf(`Error executing query: %v`, err)
		return nil, err
	}
	defer rows.Close()

	var users []user.User
	for rows.Next() {
		var user user.User

		if err := rows.Scan(&user.ID, &user.FirstName, &user.LastName, &user.Email, &user.Password,
			&user.CreatedAt, &user.UpdatedAt, &user.DeletedAt, &user.IsPremium, &user.Avatar, &user.Username); err != nil {
			log.Printf(`Error scanning user: %v`, err)
			return nil, err
		}

		users = append(users, user)
	}

	return users, nil
}

func (ur *userRepository) GetUserByID(id string) (*user.User, error) {
	// First log the query we're about to execute for debugging
	log.Printf("GetUserByID: Executing query for ID: %s", id)

	row := ur.db.QueryRow("SELECT id, first_name, last_name, email, password, created_at, updated_at, deleted_at, is_premium, avatar, username FROM users WHERE id = ?", id)

	var user user.User

	if err := row.Scan(&user.ID, &user.FirstName, &user.LastName, &user.Email, &user.Password,
		&user.CreatedAt, &user.UpdatedAt, &user.DeletedAt, &user.IsPremium, &user.Avatar, &user.Username); err != nil {
		log.Printf("Error scanning user by ID %s: %v", id, err)
		return nil, err
	}

	log.Printf(" ID=%s, Username=%s", user.ID, user.Username)
	return &user, nil
}

func (ur *userRepository) CheckPremiumByUsernameOrEmail(inputs string) (*user.User, error) {
	row := ur.db.QueryRow("SELECT is_premium FROM users WHERE username = ? OR email = ?", inputs, inputs)

	var user user.User
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

func (ur *userRepository) GetUserByEmailOrUsername(input string) (*user.User, error) {
	// Check if it's an email or username
	row := ur.db.QueryRow("SELECT id, email, username FROM users WHERE email = ? OR username = ?", input, input)

	var user user.User
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

func (ur *userRepository) CreateUser(user *user.User) error {
	// Hash the password before storing
	hashedPassword, err := utils.HashPassword(user.Password)
	if err != nil {
		log.Printf("CreateUser: Error hashing password: %v", err)
		return err
	}

	now := time.Now()
	u := uuid.Must(uuid.NewRandom())
	user.ID = u.String() // Set the generated ID back to the user struct

	_, err = ur.db.Exec("INSERT INTO users (id, first_name, last_name, email, password, created_at, updated_at, deleted_at, is_premium, avatar, username) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		user.ID, user.FirstName, user.LastName, user.Email, hashedPassword, now, now, nil, user.IsPremium, user.Avatar, user.Username)
	if err != nil {
		log.Printf("CreateUser error: %v", err)
	}
	return err
}

func (ur *userRepository) UpdateUser(id string, user *user.UpdateUser) error {
	// First, get the current user data to compare
	currentUser, err := ur.GetUserByID(id)
	if err != nil {
		log.Printf("UpdateUser: Error getting current user: %v", err)
		return err
	}

	// Build dynamic update query based on what's changed
	/*
		setParts: slice of string yang akan menampung potongan–potongan SQL SET, misalnya "first_name = ?".
	*/
	var setParts []string
	/*
		args: slice of interface{} (empty interface) — aliasnya “any type” — yang akan menampung nilai-nilai parameter untuk setiap ? di query.
		— Karena kita tidak tahu tipe apa saja yang akan dikirim (string, time.Time, bool, dll), kita gunakan interface{} agar bisa menampung semuanya.
	*/
	var args []interface{}

	// Check each field and only update if different
	if user.FirstName != currentUser.FirstName {
		setParts = append(setParts, "first_name = ?")
		args = append(args, user.FirstName)
	}

	if user.LastName != currentUser.LastName {
		setParts = append(setParts, "last_name = ?")
		args = append(args, user.LastName)
	}

	// Only update email if it's different from current email
	if user.Email != "" && user.Email != currentUser.Email {
		// Check if the new email already exists for another user
		existingUser, err := ur.GetUserByEmailOrUsername(user.Email)
		if err == nil && existingUser != nil && existingUser.ID != id {
			// Email already exists for another user
			return fmt.Errorf("email already exists")
		}
		setParts = append(setParts, "email = ?")
		args = append(args, user.Email)
	}

	if user.Avatar != "" && user.Avatar != currentUser.Avatar {
		setParts = append(setParts, "avatar = ?")
		args = append(args, user.Avatar)
	}

	// Handle password separately since it needs hashing
	if user.Password != "" {
		hashedPassword, err := utils.HashPassword(user.Password)
		if err != nil {
			log.Printf("UpdateUser: Error hashing password: %v", err)
			return err
		}
		setParts = append(setParts, "password = ?")
		args = append(args, hashedPassword)
	}

	// If no fields to update, return early
	if len(setParts) == 0 {
		log.Printf("UpdateUser: No fields to update for user ID %s", id)
		return nil
	}

	// Always update the updated_at timestamp when there are changes
	setParts = append(setParts, "updated_at = ?")
	args = append(args, time.Now())

	// Add the WHERE clause parameter
	args = append(args, id)

	/*
		Ringkasnya tentang append dan []interface{}
		append(slice, value)
		– Menambahkan value di ujung slice.
		– Jika kapasitas (cap) tidak cukup, Go membuat backing array baru di belakang layar.
		– Oleh karena itu kita tulis slice = append(slice, value).

		interface{}
		– Tipe kosong yang bisa memuat apa pun (int, string, struct, time.Time, dll).
		– Berguna untuk kumpulkan argumen dinamis ketika memanggil fungsi variadik seperti Exec(query, args ...).
	*/

	// Build the final query - manually join the setParts
	query := "UPDATE users SET "
	for i, part := range setParts {
		if i > 0 {
			query += ", "
		}
		query += part
	}
	query += " WHERE id = ?"

	_, err = ur.db.Exec(query, args...)
	if err != nil {
		log.Printf("UpdateUser error for ID %s: %v", id, err)
	}
	return err
}

func (ur *userRepository) GetUserForLogin(emailOrUsername string) (*user.User, error) {
	log.Printf("GetUserForLogin: Attempting login for: %s", emailOrUsername)

	row := ur.db.QueryRow(`
		SELECT id, username, first_name, last_name, email, password, 
		       created_at, updated_at, deleted_at, is_premium, avatar 
		FROM users 
		WHERE (email = ? OR username = ?) AND deleted_at IS NULL`,
		emailOrUsername, emailOrUsername)

	var user user.User
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

// GetAllUsersWithPagination retrieves all users with pagination (admin only)
func (ur *userRepository) GetAllUsersWithPagination(limit, offset int) ([]user.User, int64, error) {
	// Get total count
	var totalCount int64
	row := ur.db.QueryRow("SELECT COUNT(*) FROM users WHERE deleted_at IS NULL")
	if err := row.Scan(&totalCount); err != nil {
		log.Printf("GetAllUsersWithPagination: Error getting total count: %v", err)
		return nil, 0, err
	}

	// Get users with pagination
	query := `SELECT id, first_name, last_name, email, password, created_at, updated_at, 
	          deleted_at, is_premium, avatar, username, COALESCE(is_locked, false), 
	          locked_at, COALESCE(locked_reason, ''), COALESCE(role, 'user')
	          FROM users WHERE deleted_at IS NULL 
	          ORDER BY created_at DESC LIMIT ? OFFSET ?`

	rows, err := ur.db.Query(query, limit, offset)
	if err != nil {
		log.Printf("GetAllUsersWithPagination: Error executing query: %v", err)
		return nil, 0, err
	}
	defer rows.Close()

	var users []user.User
	for rows.Next() {
		var user user.User
		if err := rows.Scan(&user.ID, &user.FirstName, &user.LastName, &user.Email, &user.Password,
			&user.CreatedAt, &user.UpdatedAt, &user.DeletedAt, &user.IsPremium, &user.Avatar,
			&user.Username, &user.IsLocked, &user.LockedAt, &user.LockedReason, &user.Role); err != nil {
			log.Printf("GetAllUsersWithPagination: Error scanning user: %v", err)
			return nil, 0, err
		}
		users = append(users, user)
	}

	return users, totalCount, nil
}

// LockUser locks a user account with a reason
func (ur *userRepository) LockUser(userID, reason string) error {
	now := time.Now()
	_, err := ur.db.Exec(`UPDATE users SET is_locked = ?, locked_at = ?, locked_reason = ?, updated_at = ? 
	                      WHERE id = ? AND deleted_at IS NULL`,
		true, now, reason, now, userID)
	if err != nil {
		log.Printf("LockUser: Error locking user %s: %v", userID, err)
	}
	return err
}

// UnlockUser unlocks a user account
func (ur *userRepository) UnlockUser(userID, reason string) error {
	now := time.Now()
	unlockReason := "Account unlocked"
	if reason != "" {
		unlockReason = reason
	}

	_, err := ur.db.Exec(`UPDATE users SET is_locked = ?, locked_at = NULL, locked_reason = ?, updated_at = ? 
	                      WHERE id = ? AND deleted_at IS NULL`,
		false, unlockReason, now, userID)
	if err != nil {
		log.Printf("UnlockUser: Error unlocking user %s: %v", userID, err)
	}
	return err
}

// IsUserLocked checks if a user account is locked
func (ur *userRepository) IsUserLocked(userID string) (bool, error) {
	var isLocked bool
	row := ur.db.QueryRow("SELECT COALESCE(is_locked, false) FROM users WHERE id = ? AND deleted_at IS NULL", userID)
	err := row.Scan(&isLocked)
	if err != nil {
		log.Printf("IsUserLocked: Error checking lock status for user %s: %v", userID, err)
		return false, err
	}
	return isLocked, nil
}
