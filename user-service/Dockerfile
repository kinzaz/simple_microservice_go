# Используем базовый образ Go
FROM golang:1.23.2

# Устанавливаем рабочую директорию
WORKDIR /app

# Копируем исходный код
COPY . .

# Компилируем приложение
RUN go build -o user-service

# Запускаем приложение
CMD ["./user-service"]