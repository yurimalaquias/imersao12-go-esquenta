package main

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/go-chi/chi/v5"
	_ "github.com/go-sql-driver/mysql"
	"github.com/yurimalaquias/imersao12-go-esquenta/internal/infra/akafka"
	"github.com/yurimalaquias/imersao12-go-esquenta/internal/infra/repository"
	"github.com/yurimalaquias/imersao12-go-esquenta/internal/infra/web"
	"github.com/yurimalaquias/imersao12-go-esquenta/internal/usecase"
)

func main() {
	db, err := sql.Open("mysql", "root:root@tcp(host.docker.internal:3306)/products")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	repository := repository.NewProductRepositoryMysql(db)
	createProductsUseCase := usecase.NewCreateProductsUseCase(repository)
	listProductsUseCase := usecase.NewListProductsUseCase(repository)

	productHandlers := web.NewProductHandlers(createProductsUseCase, listProductsUseCase)

	r := chi.NewRouter()
	r.Post("/products", productHandlers.CreateProductHandler)
	r.Get("/products", productHandlers.ListProductsHandler)

	// Sobe o servidor WEB
	go http.ListenAndServe(":8000", r)

	// Comunica e sobe o Kafka
	msgChan := make(chan *kafka.Message)
	go akafka.Consume([]string{"product"}, "host.docker.internal:9094", msgChan)

	// Fica consumindo o KAFKA
	for msg := range msgChan {
		dto := usecase.CreateProductInputDto{}
		err := json.Unmarshal(msg.Value, &dto)
		if err != nil {
			// logar o erro
			continue
		}
		_, err = createProductsUseCase.Execute(dto)
	}

}
