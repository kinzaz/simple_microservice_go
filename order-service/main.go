package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/streadway/amqp"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Модель Order
type Order struct {
	ID     uint    `json:"id" gorm:"primaryKey"`
	UserID uint    `json:"user_id"`
	Amount float64 `json:"amount"`
	Status string  `json:"status"` // Например, "created"
}

// Структура User
type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func connectRabbitMQ() (*amqp.Connection, *amqp.Channel) {
	conn, err := amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}

	// Создание очереди
	_, err = ch.QueueDeclare(
		"payment_queue", // Имя очереди
		false,           // Долговечная ли очередь
		false,           // Удаляется ли очередь при неиспользовании
		false,           // Эксклюзивность
		false,           // Автоудаление
		nil,             // Аргументы
	)
	if err != nil {
		log.Fatalf("Failed to declare a queue: %v", err)
	}

	return conn, ch
}

// Настройка подключения к базе данных и миграция
func setupDatabase() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("orders.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	// Миграция таблицы Order
	db.AutoMigrate(&Order{})
	return db
}

// Создание заказа
func createOrder(w http.ResponseWriter, r *http.Request, db *gorm.DB) {
	var order Order
	err := json.NewDecoder(r.Body).Decode(&order)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Получаем пользователя по ID
	user, err := getUserByID(order.UserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Создаем заказ и сохраняем его в базе данных
	order.Status = "created" // Статус заказа по умолчанию

	if err := db.Create(&order).Error; err != nil {
		http.Error(w, "Error saving order", http.StatusInternalServerError)
		return
	}

	conn, ch := connectRabbitMQ()
	defer conn.Close()
	defer ch.Close()

	body, _ := json.Marshal(order)
	err = ch.Publish(
		"",              // exchange
		"payment_queue", // routing key
		false,           // mandatory
		false,           // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
	if err != nil {
		http.Error(w, "Failed to publish message", http.StatusInternalServerError)
		return
	}

	// Ответ с созданным заказом
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "Order for user %s created successfully with amount %.2f", user.Name, order.Amount)
}

// Получение пользователя по ID
func getUserByID(userID uint) (*User, error) {
	resp, err := http.Get(fmt.Sprintf("http://user-service:8081/users/%d", userID))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user not found")
	}

	var user User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode user response: %v", err)
	}

	return &user, nil
}

// Получение всех заказов
func getOrders(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var orders []Order
		if err := db.Find(&orders).Error; err != nil {
			http.Error(w, "Error retrieving orders", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(orders)
	}
}

func main() {
	// Настройка базы данных для заказов
	db := setupDatabase()

	// Обработчики маршрутов
	http.HandleFunc("/orders/create", func(w http.ResponseWriter, r *http.Request) {
		createOrder(w, r, db)
	})
	http.HandleFunc("/orders", func(w http.ResponseWriter, r *http.Request) {
		getOrders(db)(w, r)
	})

	log.Println("OrderService is running on port 8082")
	log.Fatal(http.ListenAndServe(":8082", nil))
}
