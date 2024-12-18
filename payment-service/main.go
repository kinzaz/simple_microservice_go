package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/streadway/amqp"
)

// Order структура для заказа
type Order struct {
	ID     uint    `json:"id"`
	UserID uint    `json:"user_id"`
	Amount float64 `json:"amount"`
}

// Обработчик сообщений в payment-service
func connectRabbitMQ() (*amqp.Connection, *amqp.Channel) {
	conn, err := amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}

	// Создание очереди перед подпиской
	_, err = ch.QueueDeclare(
		"payment_queue", // Имя очереди
		false,           // Долговечность очереди
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

// Обработчик сообщений
func processPayment(order Order) {
	fmt.Printf("Processing payment for Order ID: %d, Amount: %.2f\n", order.ID, order.Amount)
	time.Sleep(2 * time.Second) // Имитация обработки платежа
	fmt.Printf("Payment completed for Order ID: %d\n", order.ID)
}

func main() {
	conn, ch := connectRabbitMQ()
	defer conn.Close()
	defer ch.Close()

	msgs, err := ch.Consume(
		"payment_queue", // queue
		"",              // consumer
		true,            // auto-ack
		false,           // exclusive
		false,           // no-local
		false,           // no-wait
		nil,             // args
	)
	if err != nil {
		log.Fatalf("Failed to register a consumer: %v", err)
	}

	// Обрабатываем сообщения
	forever := make(chan bool)
	go func() {
		for d := range msgs {
			var order Order
			if err := json.Unmarshal(d.Body, &order); err != nil {
				log.Printf("Error decoding message: %v", err)
				continue
			}
			processPayment(order)
		}
	}()

	log.Println("PaymentService is waiting for messages...")
	<-forever
}
