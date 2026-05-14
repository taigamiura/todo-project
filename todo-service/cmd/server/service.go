package main

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
)

var errNotFound = errors.New("not found")

func fetchTodo(ctx context.Context, pool todoDB, id string, userID string) (todo, error) {
	var item todo
	err := pool.QueryRow(ctx,
		`SELECT id, title, description, completed, created_at, updated_at
     FROM todos WHERE id = $1 AND user_id = $2`,
		id,
		strings.TrimSpace(userID),
	).Scan(&item.ID, &item.Title, &item.Description, &item.Completed, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return todo{}, errNotFound
	}

	return item, nil
}

func listTodos(ctx context.Context, pool todoDB, userID string) ([]todo, error) {
	rows, err := pool.Query(ctx,
		`SELECT id, title, description, completed, created_at, updated_at
         FROM todos WHERE user_id = $1 ORDER BY updated_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	todos := make([]todo, 0)
	for rows.Next() {
		var item todo
		if err := rows.Scan(&item.ID, &item.Title, &item.Description, &item.Completed, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		todos = append(todos, item)
	}

	return todos, nil
}

func createTodo(ctx context.Context, pool todoDB, userID string, input todoInput) (todo, error) {
	item := todo{
		ID:          uuid.NewString(),
		Title:       strings.TrimSpace(input.Title),
		Description: strings.TrimSpace(input.Description),
		Completed:   input.Completed,
	}

	err := pool.QueryRow(ctx,
		`INSERT INTO todos (id, user_id, title, description, completed)
         VALUES ($1, $2, $3, $4, $5)
         RETURNING created_at, updated_at`,
		item.ID, userID, item.Title, item.Description, item.Completed,
	).Scan(&item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return todo{}, err
	}

	return item, nil
}

func updateTodo(ctx context.Context, pool todoDB, id string, userID string, input todoInput) (todo, error) {
	var item todo
	err := pool.QueryRow(ctx,
		`UPDATE todos
             SET title = $1, description = $2, completed = $3, updated_at = NOW()
           WHERE id = $4 AND user_id = $5
           RETURNING id, title, description, completed, created_at, updated_at`,
		strings.TrimSpace(input.Title),
		strings.TrimSpace(input.Description),
		input.Completed,
		id,
		userID,
	).Scan(&item.ID, &item.Title, &item.Description, &item.Completed, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return todo{}, err
	}

	return item, nil
}

func deleteTodo(ctx context.Context, pool todoDB, id string, userID string) (bool, error) {
	commandTag, err := pool.Exec(ctx,
		`DELETE FROM todos WHERE id = $1 AND user_id = $2`,
		id,
		userID,
	)
	if err != nil {
		return false, err
	}

	return commandTag.RowsAffected() > 0, nil
}

func validateTodoInput(input todoInput) error {
	title := strings.TrimSpace(input.Title)
	description := strings.TrimSpace(input.Description)

	switch {
	case title == "":
		return errors.New("タイトルは必須です。")
	case len(title) > 50:
		return errors.New("タイトルは50文字以内で入力してください。")
	case len(description) > 300:
		return errors.New("説明は300文字以内で入力してください。")
	default:
		return nil
	}
}