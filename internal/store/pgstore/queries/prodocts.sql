-- name: CreateProduct :one
INSERT INTO products (
    "seller_id", "name", "description", "base_price_cents", "auction_end"
) VALUES ($1, $2 , $3, $4, $5)
RETURNING id;