package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Структура User
type User struct {
	ID   uint   `json:"id" gorm:"primaryKey"`
	Name string `json:"name"`
}

// Подключение к базе данных и автоматические миграции
func setupDatabase() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("users.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	// Автоматически создаём таблицу пользователей
	db.AutoMigrate(&User{})
	return db
}

// Получение списка пользователей из базы данных
func getUsers(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var users []User
		if err := db.Find(&users).Error; err != nil {
			http.Error(w, "Error retrieving users", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(users)
	}
}

// Получение пользователя по ID
func getUserByID(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Извлекаем ID из URL
		idStr := r.URL.Path[len("/users/"):]
		id, err := strconv.Atoi(idStr)
		if err != nil || id <= 0 {
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}

		// Поиск пользователя в базе данных
		var user User
		if err := db.First(&user, id).Error; err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		// Отправляем найденного пользователя
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	}
}

// Добавление нового пользователя в базу данных
func addUser(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var user User
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			http.Error(w, "Invalid input", http.StatusBadRequest)
			return
		}
		if err := db.Create(&user).Error; err != nil {
			http.Error(w, "Error saving user", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		fmt.Fprintf(w, "User created with ID: %d", user.ID)
	}
}

func main() {
	// Настроить и подключиться к базе данных
	db := setupDatabase()

	// Определить маршруты
	http.HandleFunc("/users", getUsers(db))
	http.HandleFunc("/users/add", addUser(db))
	http.HandleFunc("/users/", getUserByID(db)) // Обработчик для получения пользователя по ID

	// Запуск сервера
	log.Println("UserService is running on port 8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
