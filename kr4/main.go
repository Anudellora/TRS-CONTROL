package main

import (
	"log"

	"kr4/db"
	"kr4/handlers"
)

func main() {
	database, err := db.InitDB("products.db")
	if err != nil {
		log.Fatalf("Ошибка инициализации БД: %v", err)
	}

	// Task 9.1, шаг 5: добавляем 2 начальных продукта если БД пустая
	db.Seed(database)

	r := handlers.NewRouter(database)

	log.Println("Сервер запущен на :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}
