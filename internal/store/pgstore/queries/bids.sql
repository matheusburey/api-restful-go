-- name: CreateBid :one
INSERT INTO bids ("product_id", "user_id", "amount_cents")
VALUES ($1, $2, $3)
RETURNING id;

-- name: GetBidsByProductID :many
SELECT *
FROM bids
WHERE product_id = $1
ORDER BY amount_cents DESC;

-- name: GetHighestBidByProductID :one
SELECT *
FROM bids
WHERE product_id = $1
ORDER BY amount_cents DESC;